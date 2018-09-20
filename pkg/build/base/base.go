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
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/build/base/sources"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// DefaultImageName is the default name of the built base image
const DefaultImageName = "kind-base"

// BuildContext is used to build the kind node base image, and contains
// build configuration
type BuildContext struct {
	sourceDir string
	imageTag  string
	goCmd     string
	arch      string
}

// Option is BuildContext configuration option supplied to NewBuildContext
type Option func(*BuildContext)

// WithSourceDir configures a NewBuildContext to use the source dir `sourceDir`
func WithSourceDir(sourceDir string) Option {
	return func(b *BuildContext) {
		b.sourceDir = sourceDir
	}
}

// WithImageTag configures a NewBuildContext to tag the built image with `tag`
func WithImageTag(tag string) Option {
	return func(b *BuildContext) {
		b.imageTag = tag
	}
}

// NewBuildContext creates a new BuildContext with
// default configuration
func NewBuildContext(options ...Option) *BuildContext {
	ctx := &BuildContext{
		imageTag: DefaultImageName,
		goCmd:    "go",
		arch:     "amd64",
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
	// if SourceDir is unset, use the baked in sources
	buildDir := tmpDir
	if c.sourceDir == "" {
		// populate with image sources
		err = sources.RestoreAssets(buildDir, "images/base")
		if err != nil {
			return err
		}
		buildDir = filepath.Join(buildDir, "images", "base")

	} else {
		err = fs.Copy(c.sourceDir, buildDir)
		if err != nil {
			log.Errorf("failed to copy sources to build dir %v", err)
			return err
		}
	}

	log.Infof("Building base image in: %s", buildDir)

	// build the entrypoint binary first
	if err := c.buildEntrypoint(buildDir); err != nil {
		return err
	}

	// then the actual docker image
	return c.buildImage(buildDir)
}

// builds the entrypoint binary
func (c *BuildContext) buildEntrypoint(dir string) error {
	// NOTE: this binary only uses the go1 stdlib, and is a single file
	entrypointSrc := filepath.Join(dir, "entrypoint", "main.go")
	entrypointDest := filepath.Join(dir, "entrypoint", "entrypoint")

	cmd := exec.Command(c.goCmd, "build", "-o", entrypointDest, entrypointSrc)
	// TODO(bentheelder): we may need to map between docker image arch and GOARCH
	cmd.Env = []string{"GOOS=linux", "GOARCH=" + c.arch}

	// actually build
	log.Info("Building entrypoint binary ...")
	cmd.Debug = true
	cmd.InheritOutput = true
	if err := cmd.Run(); err != nil {
		log.Errorf("Entrypoint build Failed! %v", err)
		return err
	}
	log.Info("Entrypoint build completed.")
	return nil
}

func (c *BuildContext) buildImage(dir string) error {
	// build the image, tagged as tagImageAs, using the our tempdir as the context
	cmd := exec.Command("docker", "build", "-t", c.imageTag, dir)
	cmd.Debug = true
	cmd.InheritOutput = true

	log.Info("Starting Docker build ...")
	err := cmd.Run()
	if err != nil {
		log.Errorf("Docker build Failed! %v", err)
		return err
	}
	log.Info("Docker build completed.")
	return nil
}
