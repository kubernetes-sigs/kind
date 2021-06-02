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

package addimage

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/build/addimage/internal/container/docker"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
	"sigs.k8s.io/kind/pkg/log"
)

// buildContext is used to build the kind node image, and contains
// build configuration
type buildContext struct {
	// option fields
	image            string
	baseImage        string
	additionalImages []string
	logger           log.Logger
	arch             string
}

// Build builds the cluster node image, the sourcedir must be set on
// the buildContext
func (c *buildContext) Build() (err error) {
	return c.addImages()
}

func (c *buildContext) addImages() error {

	c.logger.V(0).Info("Starting to add images to base image")
	// pull images to local docker
	for _, imageName := range c.additionalImages {
		// Check to see if the image exists and if not pull it
		_, err := docker.ImageID(imageName)
		if err != nil {
			err = docker.Pull(c.logger, imageName, dockerBuildOsAndArch(c.arch), 3)
			if err != nil {
				c.logger.Errorf("Add image build Failed! Failed to pull image %v: %v", imageName, err)
			}
		}
	}

	// create build container
	c.logger.V(0).Info("Creating build container based on " + c.baseImage)
	// pull images to local docker
	containerID, err := c.createBuildContainer()
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

	// Tar up the images to make the load easier (and follow the current load pattern)
	// Setup the tar path where the images will be saved
	dir, err := fs.TempDir("", "images-tar")
	if err != nil {
		return errors.Wrap(err, "failed to create tempdir")
	}
	defer os.RemoveAll(dir)
	imagesTarFile := filepath.Join(dir, "images.tar")
	// Save the images into a tar file
	c.logger.V(0).Info("Saving images into tar file at " + imagesTarFile)
	err = docker.SaveImages(c.additionalImages, imagesTarFile)
	if err != nil {
		return err
	}

	// setup image importer
	cmder := docker.ContainerCmder(containerID)
	importer := newContainerdImporter(cmder)

	f, err := os.Open(imagesTarFile)
	if err != nil {
		return err
	}
	defer f.Close()
	//return importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stderr).SetStdin(f).Run()
	// we will rewrite / correct the tags as we load the image
	c.logger.V(0).Info("Importing images into build container " + containerID)
	if err := exec.RunWithStdinWriter(importer.LoadCommand().SetStdout(os.Stdout).SetStderr(os.Stdout), func(w io.Writer) error {
		return docker.EditArchive(f, w, fixRepository, c.arch)
	}); err != nil {
		return err
	}

	// Save the image changes to a new image
	c.logger.V(0).Info("Saving new image " + c.image)
	saveCmd := exec.Command(
		"docker", "commit",
		// we need to put this back after changing it when running the image
		"--change", `ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]`,
		containerID, c.image,
	)
	exec.InheritOutput(saveCmd)
	if err = saveCmd.Run(); err != nil {
		c.logger.Errorf("Add image build Failed! Failed to save image: %v", err)
		return err
	}

	c.logger.V(0).Info("Add image build completed.")
	return nil
}

func (c *buildContext) createBuildContainer() (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_ = docker.Pull(c.logger, c.baseImage, dockerBuildOsAndArch(c.arch), 4)
	// this should be good enough: a specific prefix, the current unix time,
	// and a little random bits in case we have multiple builds simultaneously
	random := rand.New(rand.NewSource(time.Now().UnixNano())).Int31()
	id = fmt.Sprintf("kind-build-%d-%d", time.Now().UTC().Unix(), random)
	err = docker.Run(
		c.baseImage,
		[]string{
			"-d", // make the client exit while the container continues to run
			// run containerd so that the cri command works
			"--entrypoint=/usr/local/bin/containerd",
			"--name=" + id,
			"--platform=" + dockerBuildOsAndArch(c.arch),
		},
		[]string{
			"",
		},
	)
	if err != nil {
		return id, errors.Wrap(err, "failed to create build container")
	}
	return id, nil
}

func dockerBuildOsAndArch(arch string) string {
	return "linux/" + arch
}
