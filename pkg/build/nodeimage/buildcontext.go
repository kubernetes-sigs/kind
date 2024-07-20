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
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/container/docker"
	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/kube"
	"sigs.k8s.io/kind/pkg/internal/sets"
	"sigs.k8s.io/kind/pkg/internal/version"
)

const (
	// httpProxy is the HTTP_PROXY environment variable key
	httpProxy = "HTTP_PROXY"
	// httpsProxy is the HTTPS_PROXY environment variable key
	httpsProxy = "HTTPS_PROXY"
	// noProxy is the NO_PROXY environment variable key
	noProxy = "NO_PROXY"
)

// buildContext is used to build the kind node image, and contains
// build configuration
type buildContext struct {
	// option fields
	image     string
	baseImage string
	logger    log.Logger
	arch      string
	buildType string
	kubeParam string
	// non-option fields
	builder kube.Builder
}

// Build builds the cluster node image, the source dir must be set on
// the buildContext
func (c *buildContext) Build() (err error) {
	// ensure kubernetes build is up-to-date first
	c.logger.V(0).Info("Starting to build Kubernetes")
	bits, err := c.builder.Build()
	if err != nil {
		c.logger.Errorf("Failed to build Kubernetes: %v", err)
		return errors.Wrap(err, "failed to build kubernetes")
	}
	c.logger.V(0).Info("Finished building Kubernetes")

	// then perform the actual docker image build
	c.logger.V(0).Info("Building node image ...")
	return c.buildImage(bits)
}

func (c *buildContext) buildImage(bits kube.Bits) error {
	// create build container
	// NOTE: we are using docker run + docker commit, so we can install
	// debian packages without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	// This also allows the KubeBit implementations to programmatically
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

	c.logger.V(0).Info("Building in container: " + containerID)

	// copy artifacts in
	for _, binary := range bits.BinaryPaths() {
		// TODO: probably should be /usr/local/bin, but the existing kubelet
		// service file expects /usr/bin/kubelet
		nodePath := "/usr/bin/" + path.Base(binary)
		if err := exec.Command("docker", "cp", binary, containerID+":"+nodePath).Run(); err != nil {
			return err
		}
		if err := cmder.Command("chmod", "+x", nodePath).Run(); err != nil {
			return err
		}
		if err := cmder.Command("chown", "root:root", nodePath).Run(); err != nil {
			return err
		}
	}

	// write version
	// TODO: support grabbing version from a binary instead?
	// This may or may not be a good idea ...
	rawVersion := bits.Version()
	parsedVersion, err := version.ParseSemantic(rawVersion)
	if err != nil {
		return errors.Wrap(err, "invalid Kubernetes version")
	}
	if err := createFile(cmder, "/kind/version", rawVersion); err != nil {
		return err
	}

	// pre-pull images that were not part of the build and write CNI / storage
	// manifests
	if _, err = c.prePullImagesAndWriteManifests(bits, parsedVersion, containerID); err != nil {
		c.logger.Errorf("Image build Failed! Failed to pull Images: %v", err)
		return err
	}

	// Save the image changes to a new image
	if err = exec.Command(
		"docker", "commit",
		// we need to put this back after changing it when running the image
		"--change", `ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]`,
		// remove proxy settings since they're for the building process
		// and should not be carried with the built image
		"--change", `ENV HTTP_PROXY="" HTTPS_PROXY="" NO_PROXY=""`,
		containerID, c.image,
	).Run(); err != nil {
		c.logger.Errorf("Image build Failed! Failed to save image: %v", err)
		return err
	}

	c.logger.V(0).Infof("Image %q build completed.", c.image)
	return nil
}

// returns a set of image tags that will be side-loaded
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
func (c *buildContext) prePullImagesAndWriteManifests(bits kube.Bits, parsedVersion *version.Version, containerID string) ([]string, error) {
	// first get the images we actually built
	builtImages, err := c.getBuiltImages(bits)
	if err != nil {
		c.logger.Errorf("Image build Failed! Failed to get built images: %v", err)
		return nil, err
	}

	// helpers to run things in the build container
	cmder := docker.ContainerCmder(containerID)

	// For kubernetes v1.15+ (actually 1.16 alpha versions) we may need to
	// drop the arch suffix from images to get the expected image
	archSuffix := "-" + c.arch
	fixRepository := func(repository string) string {
		if strings.HasSuffix(repository, archSuffix) {
			fixed := strings.TrimSuffix(repository, archSuffix)
			c.logger.V(1).Info("fixed: " + repository + " -> " + fixed)
			repository = fixed
		}
		return repository
	}

	// Determine accurate built tags using the logic that will be applied
	// when rewriting tags during archive loading
	fixedImages := sets.NewString()
	fixedImagesMap := make(map[string]string, builtImages.Len()) // key: original images, value: fixed images
	for _, image := range builtImages.List() {
		registry, tag, err := docker.SplitImage(image)
		if err != nil {
			return nil, err
		}
		registry = fixRepository(registry)
		fixedImage := registry + ":" + tag
		fixedImages.Insert(fixedImage)
		fixedImagesMap[image] = fixedImage
	}
	builtImages = fixedImages
	c.logger.V(1).Info("Detected built images: " + strings.Join(builtImages.List(), ", "))

	// gets the list of images required by kubeadm
	requiredImages, err := exec.OutputLines(cmder.Command(
		"kubeadm", "config", "images", "list", "--kubernetes-version", bits.Version(),
	))
	if err != nil {
		return nil, err
	}

	// replace pause image with our own
	containerdConfig, err := exec.Output(cmder.Command("cat", containerdConfigPath))
	if err != nil {
		return nil, err
	}
	pauseImage, err := findSandboxImage(string(containerdConfig))
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

	if parsedVersion.LessThan(version.MustParseSemantic("v1.24.0")) {
		if err := configureContainerdSystemdCgroupFalse(cmder, string(containerdConfig)); err != nil {
			return nil, err
		}
	}

	// write the default CNI manifest
	if err := createFile(cmder, defaultCNIManifestLocation, defaultCNIManifest); err != nil {
		c.logger.Errorf("Image build Failed! Failed write default CNI Manifest: %v", err)
		return nil, err
	}
	// all builds should install the default CNI images from the above manifest currently
	requiredImages = append(requiredImages, defaultCNIImages...)

	// write the default Storage manifest
	if err := createFile(cmder, defaultStorageManifestLocation, defaultStorageManifest); err != nil {
		c.logger.Errorf("Image build Failed! Failed write default Storage Manifest: %v", err)
		return nil, err
	}
	// all builds should install the default storage driver images currently
	requiredImages = append(requiredImages, defaultStorageImages...)

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

	fns := []func() error{}
	for _, image := range requiredImages {
		image := image // https://golang.org/doc/faq#closures_and_goroutines
		fns = append(fns, func() error {
			if !builtImages.Has(image) {
				if err = importer.Pull(image, dockerBuildOsAndArch(c.arch)); err != nil {
					c.logger.Warnf("Failed to pull %s with error: %v", image, err)
					runE := exec.RunErrorForError(err)
					c.logger.Warn(string(runE.Output))
				}
			}
			return nil
		})
	}
	if err := errors.AggregateConcurrent(fns); err != nil {
		return nil, err
	}

	// create a plan of image loading
	loadFns := []func() error{}
	for _, image := range bits.ImagePaths() {
		image := image // capture loop var
		loadFns = append(loadFns, func() error {
			f, err := os.Open(image)
			if err != nil {
				return err
			}
			defer f.Close()
			return importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stderr).SetStdin(f).Run()
			// we will rewrite / correct the tags in tagFns below
		})
	}

	// run all image loading concurrently until one fails or all succeed
	if err := errors.UntilErrorConcurrent(loadFns); err != nil {
		c.logger.Errorf("Image build Failed! Failed to load images %v", err)
		return nil, err
	}

	// create a plan of image re-tagging
	tagFns := []func() error{}
	for unfixed, fixed := range fixedImagesMap {
		unfixed, fixed := unfixed, fixed // capture loop var
		if unfixed != fixed {
			tagFns = append(tagFns, func() error {
				return importer.Tag(unfixed, fixed)
			})
		}
	}

	// run all image re-tagging concurrently until one fails or all succeed
	if err := errors.UntilErrorConcurrent(tagFns); err != nil {
		c.logger.Errorf("Image build Failed! Failed to re-tag images %v", err)
		return nil, err
	}

	return importer.ListImported()
}

func (c *buildContext) createBuildContainer() (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// errors here are non-critical; we'll proceed with execution, which includes a pull operation
	_ = docker.Pull(c.logger, c.baseImage, dockerBuildOsAndArch(c.arch), 4)
	// this should be good enough: a specific prefix, the current unix time,
	// and a little random bits in case we have multiple builds simultaneously
	random := rand.New(rand.NewSource(time.Now().UnixNano())).Int31()
	id = fmt.Sprintf("kind-build-%d-%d", time.Now().UTC().Unix(), random)
	runArgs := []string{
		"-d",                 // make the client exit while the container continues to run
		"--entrypoint=sleep", // the container should hang forever, so we can exec in it
		"--name=" + id,
		"--platform=" + dockerBuildOsAndArch(c.arch),
		"--security-opt", "seccomp=unconfined",
	}
	// pass proxy settings from environment variables to the building container
	// to make them work during the building process
	for _, name := range []string{httpProxy, httpsProxy, noProxy} {
		val := os.Getenv(name)
		if val == "" {
			val = os.Getenv(strings.ToLower(name))
		}
		if val != "" {
			runArgs = append(runArgs, "--env", name+"="+val)
		}
	}
	err = docker.Run(
		c.baseImage,
		runArgs,
		[]string{
			"infinity", // sleep infinitely to keep container running indefinitely
		},
	)
	if err != nil {
		return id, errors.Wrap(err, "failed to create build container")
	}
	return id, nil
}
