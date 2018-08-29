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

package build

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/test-infra/kind/pkg/build/kube"
	"k8s.io/test-infra/kind/pkg/exec"
)

// NodeImageBuildContext is used to build the kind node image, and contains
// build configuration
type NodeImageBuildContext struct {
	ImageTag  string
	Arch      string
	BaseImage string
	KubeRoot  string
	Bits      kube.Bits
}

// NewNodeImageBuildContext creates a new NodeImageBuildContext with
// default configuration
func NewNodeImageBuildContext(mode string) (ctx *NodeImageBuildContext, err error) {
	kubeRoot := ""
	// apt should not fail on finding kube root as it does not use it
	if mode != "apt" {
		kubeRoot, err = kube.FindSource()
		if err != nil {
			return nil, fmt.Errorf("error finding kuberoot: %v", err)
		}
	}
	bits, err := kube.NewNamedBits(mode, kubeRoot)
	if err != nil {
		return nil, err
	}
	return &NodeImageBuildContext{
		ImageTag:  "kind-node",
		Arch:      "amd64",
		BaseImage: "kind-base",
		KubeRoot:  kubeRoot,
		Bits:      bits,
	}, nil
}

// Build builds the cluster node image, the sourcedir must be set on
// the NodeImageBuildContext
func (c *NodeImageBuildContext) Build() (err error) {
	// ensure kubernetes build is up to date first
	log.Infof("Starting to build Kubernetes")
	if err = c.Bits.Build(); err != nil {
		log.Errorf("Failed to build Kubernetes: %v", err)
		return errors.Wrap(err, "failed to build kubernetes")
	}
	log.Infof("Finished building Kubernetes")

	// create tempdir to build the image in
	buildDir, err := TempDir("", "kind-node-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(buildDir)

	log.Infof("Building node image in: %s", buildDir)

	// populate the kubernetes artifacts first
	if err := c.populateBits(buildDir); err != nil {
		return err
	}

	// then the perform the actual docker image build
	return c.buildImage(buildDir)
}

func (c *NodeImageBuildContext) populateBits(buildDir string) error {
	// always create bits dir
	bitsDir := path.Join(buildDir, "bits")
	if err := os.Mkdir(bitsDir, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}
	// copy all bits from their source path to where we will COPY them into
	// the dockerfile, see images/node/Dockerfile
	bitPaths := c.Bits.Paths()
	for src, dest := range bitPaths {
		realDest := path.Join(bitsDir, dest)
		if err := copyFile(src, realDest); err != nil {
			return errors.Wrap(err, "failed to copy build artifact")
		}
	}
	return nil
}

// BuildContainerLabelKey is applied to each build container
const BuildContainerLabelKey = "io.k8s.test-infra.kind-build"

// private kube.InstallContext implementation, local to the image build
type installContext struct {
	basePath    string
	containerID string
}

var _ kube.InstallContext = &installContext{}

func (ic *installContext) BasePath() string {
	return ic.basePath
}

func (ic *installContext) Run(command string, args ...string) error {
	cmd := exec.Command("docker", "exec", ic.containerID, command)
	cmd.Args = append(cmd.Args, args...)
	cmd.Debug = true
	cmd.InheritOutput = true
	return cmd.Run()
}

func (ic *installContext) CombinedOutputLines(command string, args ...string) ([]string, error) {
	cmd := exec.Command("docker", "exec", ic.containerID, command)
	cmd.Args = append(cmd.Args, args...)
	cmd.Debug = true
	return cmd.CombinedOutputLines()
}

func (c *NodeImageBuildContext) buildImage(dir string) error {
	// build the image, tagged as tagImageAs, using the our tempdir as the context
	log.Info("Starting image build ...")
	// create build container
	// NOTE: we are using docker run + docker commit so we can install
	// debians without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	// This also allows the KubeBit implementations to perform programmatic
	// isntall in the image
	containerID, err := c.createBuildContainer(dir)
	if err != nil {
		log.Errorf("Image build Failed! %v", err)
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
	if err = execInBuild("mkdir", "/kind/"); err != nil {
		log.Errorf("Image build Failed! %v", err)
		return err
	}

	// copy artifacts in
	if err = execInBuild("rsync", "-r", "/build/bits/", "/kind/"); err != nil {
		log.Errorf("Image build Failed! %v", err)
		return err
	}

	// install the kube bits
	ic := &installContext{
		basePath:    "/kind/",
		containerID: containerID,
	}
	if err = c.Bits.Install(ic); err != nil {
		log.Errorf("Image build Failed! %v", err)
		return err
	}

	// ensure we don't fail if swap is enabled on the host
	if err = execInBuild("/bin/sh", "-c",
		`echo "KUBELET_EXTRA_ARGS=--fail-swap-on=false" >> /etc/default/kubelet`,
	); err != nil {
		log.Errorf("Image build Failed! %v", err)
		return err
	}

	// Save the image changes to a new image
	cmd := exec.Command("docker", "commit", containerID, c.ImageTag)
	cmd.Debug = true
	cmd.InheritOutput = true
	if err = cmd.Run(); err != nil {
		log.Errorf("Image build Failed! %v", err)
		return err
	}

	log.Info("Image build completed.")
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
