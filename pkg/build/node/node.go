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

// Package node implements functionality to build the kind node image
package node

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/build/kube"
	"sigs.k8s.io/kind/pkg/docker"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// DefaultImageName is the default name for the built image
const DefaultImageName = "kindest/node"

// DefaultImageTag is the default tag for the built image
const DefaultImageTag = "latest"

// DefaultBaseImageName is the default base image name used
const DefaultBaseImageName = "kindest/base"

// DefaultBaseImageTag is the default base image tag used
// TODO(bentheelder): add tooling to automanage this,
// and document using --base-tag=latest for local builds
const DefaultBaseImageTag = "v20180920-afa27a7"

// DefaultMode is the default kubernetes build mode for the built image
// see pkg/build/kube.Bits
const DefaultMode = "docker"

// Option is BuildContext configuration option supplied to NewBuildContext
type Option func(*BuildContext)

// WithImageName configures a NewBuildContext to tag the built image with `name`
func WithImageName(name string) Option {
	return func(b *BuildContext) {
		b.imageName = name
	}
}

// WithImageTag configures a NewBuildContext to tag the built image with `tag`
func WithImageTag(tag string) Option {
	return func(b *BuildContext) {
		b.imageTag = tag
	}
}

// WithBaseImageName configures a NewBuildContext to use `image` as the base image name
func WithBaseImageName(image string) Option {
	return func(b *BuildContext) {
		b.baseImageName = image
	}
}

// WithBaseImageTag configures a NewBuildContext to use `tag` as the base image tag
func WithBaseImageTag(tag string) Option {
	return func(b *BuildContext) {
		b.baseImageTag = tag
	}
}

// WithMode sets the kubernetes build mode for the build context
func WithMode(mode string) Option {
	return func(b *BuildContext) {
		b.mode = mode
	}
}

// BuildContext is used to build the kind node image, and contains
// build configuration
type BuildContext struct {
	// option fields
	mode          string
	imageName     string
	imageTag      string
	baseImageName string
	baseImageTag  string
	// non-option fields
	arch      string // TODO(bentheelder): this should be an option
	image     string
	baseImage string
	kubeRoot  string
	bits      kube.Bits
}

// NewBuildContext creates a new BuildContext with default configuration,
// overridden by the options supplied in the order that they are supplied
func NewBuildContext(options ...Option) (ctx *BuildContext, err error) {
	// default options
	ctx = &BuildContext{
		mode:          DefaultMode,
		imageName:     DefaultImageName,
		imageTag:      DefaultImageTag,
		arch:          "amd64",
		baseImageName: DefaultBaseImageName,
		baseImageTag:  DefaultBaseImageTag,
	}
	// apply user options
	for _, option := range options {
		option(ctx)
	}
	// normalize name and tag into an image reference
	ctx.image = docker.JoinNameAndTag(ctx.imageName, ctx.imageTag)
	ctx.baseImage = docker.JoinNameAndTag(ctx.baseImageName, ctx.baseImageTag)
	// lookup kuberoot unless mode == "apt",
	// apt should not fail on finding kube root as it does not use it
	kubeRoot := ""
	if ctx.mode != "apt" {
		kubeRoot, err = kube.FindSource()
		if err != nil {
			return nil, fmt.Errorf("error finding kuberoot: %v", err)
		}
	}
	ctx.kubeRoot = kubeRoot
	// initialize bits
	bits, err := kube.NewNamedBits(ctx.mode, kubeRoot)
	if err != nil {
		return nil, err
	}
	ctx.bits = bits
	return ctx, nil
}

// Build builds the cluster node image, the sourcedir must be set on
// the BuildContext
func (c *BuildContext) Build() (err error) {
	// ensure kubernetes build is up to date first
	log.Infof("Starting to build Kubernetes")
	if err = c.bits.Build(); err != nil {
		log.Errorf("Failed to build Kubernetes: %v", err)
		return errors.Wrap(err, "failed to build kubernetes")
	}
	log.Infof("Finished building Kubernetes")

	// create tempdir to build the image in
	buildDir, err := fs.TempDir("", "kind-node-image")
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

func (c *BuildContext) populateBits(buildDir string) error {
	// always create bits dir
	bitsDir := path.Join(buildDir, "bits")
	if err := os.Mkdir(bitsDir, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}
	// copy all bits from their source path to where we will COPY them into
	// the dockerfile, see images/node/Dockerfile
	bitPaths := c.bits.Paths()
	for src, dest := range bitPaths {
		realDest := path.Join(bitsDir, dest)
		// NOTE: we use copy not copyfile because copy ensures the dest dir
		if err := fs.Copy(src, realDest); err != nil {
			return errors.Wrap(err, "failed to copy build artifact")
		}
	}
	return nil
}

// BuildContainerLabelKey is applied to each build container
const BuildContainerLabelKey = "io.k8s.sigs.kind.build"

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

func (c *BuildContext) buildImage(dir string) error {
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
	if err = c.bits.Install(ic); err != nil {
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
	cmd := exec.Command("docker", "commit", containerID, c.image)
	cmd.Debug = true
	cmd.InheritOutput = true
	if err = cmd.Run(); err != nil {
		log.Errorf("Image build Failed! %v", err)
		return err
	}

	log.Info("Image build completed.")
	return nil
}

func (c *BuildContext) createBuildContainer(buildDir string) (id string, err error) {
	cmd := exec.Command("docker", "run")
	cmd.Args = append(cmd.Args,
		"-d", // make the client exit while the container continues to run
		// label the container to make them easier to track
		"--label", fmt.Sprintf("%s=%s", BuildContainerLabelKey, time.Now().Format(time.RFC3339Nano)),
		"-v", fmt.Sprintf("%s:/build", buildDir),
		// the container should hang forever so we can exec in it
		"--entrypoint=sleep",
		c.baseImage, // use the selected base image
		"infinity",  // sleep infinitely
	)
	cmd.Debug = true
	lines, err := cmd.CombinedOutputLines()
	if err != nil {
		return "", errors.Wrap(err, "failed to create build container")
	}
	if len(lines) < 1 {
		return "", fmt.Errorf("invalid container creation output, must print at least one line")
	}
	return lines[len(lines)-1], nil
}
