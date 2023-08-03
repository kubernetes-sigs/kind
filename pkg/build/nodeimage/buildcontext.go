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

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/container/docker"
	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/kube"
	"sigs.k8s.io/kind/pkg/internal/sets"
	"sigs.k8s.io/kind/pkg/internal/version"
)

// buildContext is used to build the kind node image, and contains
// build configuration
type buildContext struct {
	// option fields
	image     string
	baseImage string
	logger    log.Logger
	arch      string
	kubeRoot  string
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

	// make artifacts directory
	// TODO: remove this after the next release, we pre-create this in the base image now
	if err = cmder.Command("mkdir", "-p", "/kind/").Run(); err != nil {
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
			//return importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stderr).SetStdin(f).Run()
			// we will rewrite / correct the tags as we load the image
			if err := exec.RunWithStdinWriter(importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stdout), func(w io.Writer) error {
				return docker.EditArchive(f, w, fixRepository, c.arch)
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
	// we don't care if this returns error, we'll still try to run which also pulls
	_ = docker.Pull(c.logger, c.baseImage, dockerBuildOsAndArch(c.arch), 4)
	// this should be good enough: a specific prefix, the current unix time,
	// and a little random bits in case we have multiple builds simultaneously
	random := rand.New(rand.NewSource(time.Now().UnixNano())).Int31()
	id = fmt.Sprintf("kind-build-%d-%d", time.Now().UTC().Unix(), random)
	err = docker.Run(
		c.baseImage,
		[]string{
			"-d", // make the client exit while the container continues to run
			// the container should hang forever, so we can exec in it
			"--entrypoint=sleep",
			"--name=" + id,
			"--platform=" + dockerBuildOsAndArch(c.arch),
			"--security-opt", "seccomp=unconfined", // ignore seccomp
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
