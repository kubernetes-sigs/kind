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

// Package build implements functionality to build the kind images
// TODO(bentheelder): and k8s
package build

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"k8s.io/test-infra/kind/pkg/build/sources"
	"k8s.io/test-infra/kind/pkg/exec"
)

// NodeImageBuildContext is used to build the kind node image, and contains
// build configuration
type NodeImageBuildContext struct {
	SourceDir string
	ImageTag  string
	Arch      string
	BaseImage string
}

// NewNodeImageBuildContext creates a new NodeImageBuildContext with
// default configuration
func NewNodeImageBuildContext() *NodeImageBuildContext {
	return &NodeImageBuildContext{
		ImageTag:  "kind-node",
		Arch:      "amd64",
		BaseImage: "kind-base",
	}
}

// Build builds the cluster node image, the sourcedir must be set on
// the NodeImageBuildContext
func (c *NodeImageBuildContext) Build() (err error) {
	// get k8s source
	kubeRoot, err := FindKubeSource()
	if err != nil {
		return errors.Wrap(err, "could not find kubernetes source")
	}

	// ensure kubernetes build is up to date first
	glog.Infof("Starting to build Kubernetes")
	//c.buildKube(kubeRoot)
	glog.Infof("Finished building Kubernetes")
	// TODO(bentheelder): allow other types of bits
	bits, err := NewBazelBuildBits(kubeRoot)

	// create tempdir to build in
	tmpDir, err := TempDir("", "kind-node-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// populate with image sources
	// if SourceDir is unset, use the baked in sources
	buildDir := tmpDir
	if c.SourceDir == "" {
		// populate with image sources
		err = sources.RestoreAssets(buildDir, "images/node")
		if err != nil {
			return err
		}
		buildDir = filepath.Join(buildDir, "images", "node")

	} else {
		err = copyDir(c.SourceDir, buildDir)
		if err != nil {
			glog.Errorf("failed to copy sources to build dir %v", err)
			return err
		}
	}

	glog.Infof("Building node image in: %s", buildDir)

	// populate the kubernetes artifacts first
	if err := c.populateBits(buildDir, bits); err != nil {
		return err
	}

	// then the actual docker image
	return c.buildImage(buildDir)
}

func (c *NodeImageBuildContext) buildKube(kubeRoot string) error {
	// TODO(bentheelder): support other modes of building
	// cd to k8s source
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	os.Chdir(kubeRoot)
	// make sure we cd back when done
	defer os.Chdir(cwd)

	// TODO(bentheelder): move this out and next to the KubeBits impl
	cmd := exec.Command("bazel", "build")
	cmd.Args = append(cmd.Args,
		// TODO(bentheelder): we assume linux amd64, but we could select
		// this based on Arch etc. throughout, this flag supports GOOS/GOARCH
		"--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64",
		// we want the debian packages
		"//build/debs:debs",
		// and the docker images
		"//build:docker-artifacts",
	)

	cmd.Debug = true
	cmd.InheritOutput = true
	return cmd.Run()
}

func (c *NodeImageBuildContext) populateBits(buildDir string, bits KubeBits) error {
	// copy all bits from their source path to where we will COPY them into
	// the dockerfile, see images/node/Dockerfile
	bitPaths := bits.Paths()
	for src, dest := range bitPaths {
		realDest := path.Join(buildDir, "files", dest)
		if err := copyFile(src, realDest); err != nil {
			return errors.Wrap(err, "failed to copy build artifact")
		}
	}
	return nil
}

// BuildContainerLabelKey is applied to each build container
const BuildContainerLabelKey = "io.k8s.test-infra.kind-build"

func (c *NodeImageBuildContext) buildImage(dir string) error {
	// build the image, tagged as tagImageAs, using the our tempdir as the context
	glog.Info("Starting image build ...")
	// create build container
	// NOTE: we are using docker run + docker commit so we can install
	// debians without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	containerID, err := c.createBuildContainer(dir)
	if err != nil {
		glog.Errorf("Image build Failed! %v", err)
		return err
	}

	// ensure we will delete it
	defer func() {
		exec.Command("docker", "rm", "-f", containerID).Run()
	}()

	// helper we will use to run "build steps"
	execInBuild := func(command ...string) error {
		cmd := exec.Command("docker", "exec", containerID)
		cmd.Args = append(cmd.Args, command...)
		cmd.Debug = true
		cmd.InheritOutput = true
		return cmd.Run()
	}

	// make artifacts directory
	if err = execInBuild("mkdir", "-p", "/kind/bits"); err != nil {
		glog.Errorf("Image build Failed! %v", err)
		return err
	}

	// copy artifacts in
	if err = execInBuild("rsync", "-r", "/build/files/", "/kind/bits/"); err != nil {
		glog.Errorf("Image build Failed! %v", err)
		return err
	}

	// install debs
	if err = execInBuild("/bin/sh", "-c", "dpkg -i /kind/bits/debs/*.deb"); err != nil {
		glog.Errorf("Image build Failed! %v", err)
		return err
	}

	// clean up after debs / remove them, this saves a couple hundred MB
	if err = execInBuild("/bin/sh", "-c",
		"rm -rf /kind/bits/debs/*.deb /var/cache/debconf/* /var/lib/apt/lists/* /var/log/*kg",
	); err != nil {
		glog.Errorf("Image build Failed! %v", err)
		return err
	}

	// ensure we don't fail if swap is enabled on the host
	if err = execInBuild("/bin/sh", "-c",
		`echo "KUBELET_EXTRA_ARGS=--fail-swap-on=false" >> /etc/default/kubelet`,
	); err != nil {
		glog.Errorf("Image build Failed! %v", err)
		return err
	}

	// Save the image changes to a new image
	cmd := exec.Command("docker", "commit", containerID, c.ImageTag)
	cmd.Debug = true
	cmd.InheritOutput = true
	if err = cmd.Run(); err != nil {
		glog.Errorf("Image build Failed! %v", err)
		return err
	}

	glog.Info("Image build completed.")
	return nil
}

func (c *NodeImageBuildContext) createBuildContainer(buildDir string) (id string, err error) {
	cmd := exec.Command("docker", "run")
	cmd.Args = append(cmd.Args,
		"-d", // make the client exit while the container continues to run
		// label the container to make them easier to track
		"--label", fmt.Sprintf("%s=%s", BuildContainerLabelKey, time.Now().Format(time.RFC3339Nano)),
		"-v", fmt.Sprintf("%s:/build", buildDir),
		c.BaseImage,
	)
	cmd.Debug = true
	lines, err := cmd.CombinedOutputLines()
	if err != nil {
		return "", errors.Wrap(err, "failed to create build container")
	}
	if len(lines) != 1 {
		return "", fmt.Errorf("invalid container creation output: %v", lines)
	}
	return lines[0], nil
}
