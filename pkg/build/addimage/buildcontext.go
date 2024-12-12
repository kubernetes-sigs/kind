/*
Copyrigh. 2024 The Kubernetes Authors.

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
	"math/rand"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/build/internal/build"
	"sigs.k8s.io/kind/pkg/build/internal/container/docker"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

const (
	// httpProxy is the HTTP_PROXY environment variable key
	httpProxy = "HTTP_PROXY"
	// httpsProxy is the HTTPS_PROXY environment variable key
	httpsProxy = "HTTPS_PROXY"
	// noProxy is the NO_PROXY environment variable key
	noProxy = "NO_PROXY"
)

// buildContext the settings to use for rebuilding the node image.
type buildContext struct {
	// option fields
	image            string
	baseImage        string
	additionalImages []string
	logger           log.Logger
	arch             string
}

// Build rebuilds the cluster node image using the buildContext to determine
// which base image and additional images to package into a new node image.
func (c *buildContext) Build() (err error) {
	c.logger.V(0).Infof("Adding %v images to base image", c.additionalImages)

	c.logger.V(0).Infof("Creating build container based on %q", c.baseImage)
	containerID, err := c.createBuildContainer()
	if containerID != "" {
		defer func() {
			_ = exec.Command("docker", "rm", "-f", "-v", containerID).Run()
		}()
	}
	if err != nil {
		c.logger.Errorf("add image build failed, unable to create build container: %v", err)
		return err
	}
	c.logger.V(1).Infof("Building in %s", containerID)

	// Pull the images into our build container
	cmder := docker.ContainerCmder(containerID)
	importer := build.NewContainerdImporter(cmder)

	c.logger.V(0).Infof("Importing images into build container %s", containerID)
	for _, imageName := range c.additionalImages {
		// Normalize the name to what would be expected with a `docker pull`
		if !strings.Contains(imageName, "/") {
			imageName = fmt.Sprintf("docker.io/library/%s", imageName)
		}

		if !strings.Contains(imageName, ":") {
			imageName += ":latest"
		}

		err = importer.Pull(imageName, build.DockerBuildOsAndArch(c.arch))
		if err != nil {
			c.logger.Errorf("add image build failed, unable to pull image %q: %v", imageName, err)
			return err
		}
	}

	// Save the image changes to a new image
	c.logger.V(0).Info("Saving new image " + c.image)
	saveCmd := exec.Command(
		"docker", "commit",
		// we need to put this back after changing it when running the image
		"--change", `ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]`,
		// remove proxy settings since they're for the building process
		// and should not be carried with the built image
		"--change", `ENV HTTP_PROXY="" HTTPS_PROXY="" NO_PROXY=""`,
		containerID, c.image,
	)
	exec.InheritOutput(saveCmd)
	if err = saveCmd.Run(); err != nil {
		c.logger.Errorf("add image build failed, unable to save destination image: %v", err)
		return err
	}

	c.logger.V(0).Info("Add image build completed.")
	return nil
}

func (c *buildContext) createBuildContainer() (id string, err error) {
	// Attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_ = docker.Pull(c.logger, c.baseImage, build.DockerBuildOsAndArch(c.arch), 4)

	// This should be good enough: a specific prefix, the current unix time,
	// and a little random bits in case we have multiple builds simultaneously
	random := rand.New(rand.NewSource(time.Now().UnixNano())).Int31()
	id = fmt.Sprintf("kind-build-%d-%d", time.Now().UTC().Unix(), random)
	runArgs := []string{
		// make the client exit while the container continues to run
		"-d",
		// run containerd so that the cri command works
		"--entrypoint=/usr/local/bin/containerd",
		"--name=" + id,
		"--platform=" + build.DockerBuildOsAndArch(c.arch),
		"--security-opt", "seccomp=unconfined", // ignore seccomp
	}

	// Pass proxy settings from environment variables to the building container
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

	// Run it
	err = docker.Run(
		c.baseImage,
		runArgs,
		[]string{
			"",
		},
	)

	return id, errors.Wrap(err, "failed to create build container")
}
