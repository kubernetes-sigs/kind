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

// Package base implements functionality to build the kind base image
package base

import (
	"go/build"
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
	"sigs.k8s.io/kind/pkg/log"
)

// DefaultImage is the default name:tag of the built base image
const DefaultImage = "kindest/base:latest"

// BuildContext is used to build the kind node base image, and contains
// build configuration
type BuildContext struct {
	// option fields
	sourceDir string
	image     string
	logger    log.Logger
}

// Option is BuildContext configuration option supplied to NewBuildContext
type Option func(*BuildContext)

// WithSourceDir configures a NewBuildContext to use the source dir `sourceDir`
func WithSourceDir(sourceDir string) Option {
	return func(b *BuildContext) {
		b.sourceDir = sourceDir
	}
}

// WithImage configures a NewBuildContext to tag the built image with `name`
func WithImage(image string) Option {
	return func(b *BuildContext) {
		b.image = image
	}
}

// WithLogger configures a NewBuildContext to log using logger
func WithLogger(logger log.Logger) Option {
	return func(b *BuildContext) {
		b.logger = logger
	}
}

// NewBuildContext creates a new BuildContext with
// default configuration
func NewBuildContext(options ...Option) *BuildContext {
	ctx := &BuildContext{
		image:  DefaultImage,
		logger: log.NoopLogger{},
	}
	for _, option := range options {
		option(ctx)
	}
	return ctx
}

// Build builds the cluster node image, the sourcedir must be set on
// the NodeImageBuildContext
func (c *BuildContext) Build() (err error) {
	// create tempdir to build in
	tmpDir, err := fs.TempDir("", "kind-base-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// populate with image sources
	// if SourceDir is unset then try to autodetect source dir
	buildDir := tmpDir
	if c.sourceDir == "" {
		pkg, err := build.Default.Import("sigs.k8s.io/kind", build.Default.GOPATH, build.FindOnly)
		if err != nil {
			return errors.Wrap(err, "failed to locate sources")
		}
		c.sourceDir = filepath.Join(pkg.Dir, "images", "base")
	}

	err = fs.Copy(c.sourceDir, buildDir)
	if err != nil {
		c.logger.Errorf("failed to copy sources to build dir %v", err)
		return err
	}

	c.logger.V(0).Infof("Building base image in: %s", buildDir)

	// then the actual docker image
	return c.buildImage(buildDir)
}

func (c *BuildContext) buildImage(dir string) error {
	// build the image, tagged as tagImageAs, using the our tempdir as the context
	cmd := exec.Command("docker", "build", "-t", c.image, dir)
	c.logger.V(0).Info("Starting Docker build ...")
	exec.InheritOutput(cmd)
	err := cmd.Run()
	if err != nil {
		c.logger.Errorf("Docker build Failed! %v", err)
		return err
	}
	c.logger.V(0).Info("Docker build completed.")
	return nil
}
