/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodeimage

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/version"

	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/container/docker"
	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/kube"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// Build builds the cluster node image, the sourcedir must be set on
// the buildContext
func (c *buildContext) Build() (err error) {
	// ensure kubernetes build is up to date first
	c.logger.V(0).Info("Starting to build Kubernetes")
	bits, err := c.builder.Build()
	if err != nil {
		c.logger.Errorf("Failed to build Kubernetes: %v", err)
		return errors.Wrap(err, "failed to build kubernetes")
	}
	c.logger.V(0).Info("Finished building Kubernetes")

	// then the perform the actual docker image build
	c.logger.V(0).Info("Building node image ...")
	return c.buildImage(bits)
}

func (c *buildContext) buildImage(bits kube.Bits) error {
	// create build container
	// NOTE: we are using docker run + docker commit so we can install
	// debian packages without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	// This also allows the KubeBit implementations to perform programmatic
	// install in the image
	containerID, err := c.createBuildContainer()
	cmder := docker.ContainerCmder(containerID)

	// ensure we will delete it
	if containerID != "" {
		defer func() {
			_ = exec.Command("docker", "rm", "-f", "-v", containerID).Run()
		}()
	}
	if err != nil {
		c.logger.Errorf("Image build Failed! Failed to create build container: %v", err)
		return err
	}

	c.logger.V(0).Info("Building in " + containerID)

	// helper we will use to run "build steps"
	execInBuild := func(command string, args ...string) error {
		return exec.InheritOutput(cmder.Command(command, args...)).Run()
	}

	// make artifacts directory
	if err = execInBuild("mkdir", "/kind/"); err != nil {
		c.logger.Errorf("Image build Failed! Failed to make directory %v", err)
		return err
	}

	// copy artifacts in
	for _, binary := range bits.BinaryPaths() {
		// TODO: probably should be /usr/local/bin, but the existing kubelet
		// service file expects /usr/bin/kubelet
		nodePath := "/usr/bin/" + path.Base(binary)
		if err := exec.Command("docker", "cp", binary, containerID+":"+nodePath).Run(); err != nil {
			return err
		}
		if err := execInBuild("chmod", "+x", nodePath); err != nil {
			return err
		}
		if err := execInBuild("chown", "root:root", nodePath); err != nil {
			return err
		}
	}

	// write version
	// TODO: support grabbing version from a binary instead
	if err := createFile(cmder, "/kind/version", bits.Version()); err != nil {
		return err
	}

	dir, err := fs.TempDir("", "kind-build")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	// pre-pull images that were not part of the build
	if _, err = c.prePullImages(bits, dir, containerID); err != nil {
		c.logger.Errorf("Image build Failed! Failed to pull Images: %v", err)
		return err
	}

	// Save the image changes to a new image
	cmd := exec.Command(
		"docker", "commit",
		// we need to put this back after changing it when running the image
		"--change", `ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]`,
		containerID, c.image,
	)
	exec.InheritOutput(cmd)
	if err = cmd.Run(); err != nil {
		c.logger.Errorf("Image build Failed! Failed to save image: %v", err)
		return err
	}

	c.logger.V(0).Info("Image build completed.")
	return nil
}

// returns a set of image tags that will be sideloaded
func (c *buildContext) getBuiltImages(bits kube.Bits) (sets.String, error) {
	images := sets.NewString()
	for _, path := range bits.ImagePaths() {
		tags, err := docker.GetArchiveTags(path)
		if err != nil {
			return nil, err
		}
		images.Insert(tags...)
	}
	return images, nil
}

// must be run after kubernetes has been installed on the node
func (c *buildContext) prePullImages(bits kube.Bits, dir, containerID string) ([]string, error) {
	// first get the images we actually built
	builtImages, err := c.getBuiltImages(bits)
	if err != nil {
		c.logger.Errorf("Image build Failed! Failed to get built images: %v", err)
		return nil, err
	}

	// helpers to run things in the build container
	cmder := docker.ContainerCmder(containerID)

	// get the Kubernetes version we installed on the node
	// we need this to ask kubeadm what images we need
	rawVersion, err := exec.OutputLines(cmder.Command("cat", kubernetesVersionLocation))
	if err != nil {
		c.logger.Errorf("Image build Failed! Failed to get Kubernetes version: %v", err)
		return nil, err
	}
	if len(rawVersion) != 1 {
		c.logger.Errorf("Image build Failed! Failed to get Kubernetes version: %v", err)
		return nil, errors.New("invalid kubernetes version file")
	}

	// parse version for comparison
	ver, err := version.ParseSemantic(rawVersion[0])
	if err != nil {
		return nil, err
	}

	// For kubernetes v1.15+ (actually 1.16 alpha versions) we may need to
	// drop the arch suffix from images to get the expected image
	archSuffix := "-" + c.arch
	fixRepository := func(repository string) string {
		if strings.HasSuffix(repository, archSuffix) {
			fixed := strings.TrimSuffix(repository, archSuffix)
			fmt.Println("fixed: " + repository + " -> " + fixed)
			repository = fixed
		}
		return repository
	}

	// correct set of built tags using the same logic we will use to rewrite
	// the tags as we load the archives
	fixedImages := sets.NewString()
	for _, image := range builtImages.List() {
		registry, tag, err := docker.SplitImage(image)
		if err != nil {
			return nil, err
		}
		registry = fixRepository(registry)
		fixedImages.Insert(registry + ":" + tag)
	}
	builtImages = fixedImages
	c.logger.V(0).Info("Detected built images: " + strings.Join(builtImages.List(), ", "))

	// gets the list of images required by kubeadm
	requiredImages, err := exec.OutputLines(cmder.Command(
		"kubeadm", "config", "images", "list", "--kubernetes-version", rawVersion[0],
	))
	if err != nil {
		return nil, err
	}

	// replace pause image with our own
	config, err := exec.Output(cmder.Command("cat", "/etc/containerd/config.toml"))
	if err != nil {
		return nil, err
	}
	pauseImage, err := findSandboxImage(string(config))
	if err != nil {
		return nil, err
	}
	n := 0
	for _, image := range requiredImages {
		if !strings.Contains(image, "pause") {
			requiredImages[n] = image
			n++
		}
	}
	requiredImages = append(requiredImages[:n], pauseImage)

	// write the default CNI manifest
	if err := createFile(cmder, defaultCNIManifestLocation, defaultCNIManifest); err != nil {
		c.logger.Errorf("Image build Failed! Failed write default CNI Manifest: %v", err)
		return nil, err
	}
	// all builds should install the default CNI images from the above manifest currently
	requiredImages = append(requiredImages, defaultCNIImages...)

	// write the default Storage manifest
	// in < 1.14 we need to use beta labels
	storageManifest := defaultStorageManifest
	if ver.LessThan(version.MustParseSemantic("v1.14.0")) {
		storageManifest = strings.ReplaceAll(storageManifest, "kubernetes.io/os", "beta.kubernetes.io/os")
	}
	if err := createFile(cmder, defaultStorageManifestLocation, storageManifest); err != nil {
		c.logger.Errorf("Image build Failed! Failed write default Storage Manifest: %v", err)
		return nil, err
	}
	// all builds should install the default storage driver images currently
	requiredImages = append(requiredImages, defaultStorageImages...)

	// Create "images" subdir.
	imagesDir := path.Join(dir, "bits", "images")
	if err := os.MkdirAll(imagesDir, 0777); err != nil {
		c.logger.Errorf("Image build Failed! Failed create local images dir: %v", err)
		return nil, errors.Wrap(err, "failed to make images dir")
	}

	fns := []func() error{}
	pulledImages := make(chan string, len(requiredImages))
	for i, image := range requiredImages {
		i, image := i, image // https://golang.org/doc/faq#closures_and_goroutines
		fns = append(fns, func() error {
			if !builtImages.Has(image) {
				fmt.Printf("Pulling: %s\n", image)
				err := docker.Pull(c.logger, image, 2)
				if err != nil {
					c.logger.Warnf("Failed to pull %s with error: %v", image, err)
				}
				// TODO(bentheelder): generate a friendlier name
				pullName := fmt.Sprintf("%d.tar", i)
				pullTo := path.Join(imagesDir, pullName)
				err = docker.Save(image, pullTo)
				if err != nil {
					return err
				}
				pulledImages <- pullTo
			}
			return nil
		})
	}
	if err := errors.AggregateConcurrent(fns); err != nil {
		return nil, err
	}
	close(pulledImages)
	pulled := []string{}
	for image := range pulledImages {
		pulled = append(pulled, image)
	}

	// setup image importer
	importer := newContainerdImporter(cmder)
	if err := importer.Prepare(); err != nil {
		c.logger.Errorf("Image build Failed! Failed to prepare containerd to load images %v", err)
		return nil, err
	}

	// TODO: return this error?
	defer func() {
		if err := importer.End(); err != nil {
			c.logger.Errorf("Image build Failed! Failed to tear down containerd after loading images %v", err)
		}
	}()

	// create a plan of image loading
	loadFns := []func() error{}
	for _, image := range pulled {
		image := image // capture loop var
		loadFns = append(loadFns, func() error {
			f, err := os.Open(image)
			if err != nil {
				return err
			}
			defer f.Close()
			return importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stdout).SetStdin(f).Run()
		})
	}
	for _, image := range bits.ImagePaths() {
		image := image // capture loop var
		loadFns = append(loadFns, func() error {
			f, err := os.Open(image)
			if err != nil {
				return err
			}
			defer f.Close()
			//return importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stderr).SetStdin(f).Run()
			// we will rewrite / correct the tags as we load the image
			if err := exec.RunWithStdinWriter(importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stdout), func(w io.Writer) error {
				return docker.EditArchiveRepositories(f, w, fixRepository)
			}); err != nil {
				return err
			}
			return nil
		})
	}

	// run all image loading concurrently until one fails or all succeed
	if err := errors.UntilErrorConcurrent(loadFns); err != nil {
		c.logger.Errorf("Image build Failed! Failed to load images %v", err)
		return nil, err
	}

	return importer.ListImported()
}

func (c *buildContext) createBuildContainer() (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = docker.PullIfNotPresent(c.logger, c.baseImage, 4)
	// this should be good enough: a specific prefix, the current unix time,
	// and a little random bits in case we have multiple builds simultaneously
	random := rand.New(rand.NewSource(time.Now().UnixNano())).Int31()
	id = fmt.Sprintf("kind-build-%d-%d", time.Now().UTC().Unix(), random)
	err = docker.Run(
		c.baseImage,
		[]string{
			"-d", // make the client exit while the container continues to run
			// the container should hang forever so we can exec in it
			"--entrypoint=sleep",
			"--name=" + id,
		},
		[]string{
			"infinity", // sleep infinitely to keep the container around
		},
	)
	if err != nil {
		return id, errors.Wrap(err, "failed to create build container")
	}
	return id, nil
}
