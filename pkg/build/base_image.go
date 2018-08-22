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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/glog"

	"k8s.io/test-infra/kind/pkg/build/sources"
	"k8s.io/test-infra/kind/pkg/exec"
)

// BaseImageBuildContext is used to build the kind node base image, and contains
// build configuration
type BaseImageBuildContext struct {
	SourceDir string
	ImageTag  string
	GoCmd     string
	Arch      string
}

// NewBaseImageBuildContext creates a new BaseImageBuildContext with
// default configuration
func NewBaseImageBuildContext() *BaseImageBuildContext {
	return &BaseImageBuildContext{
		ImageTag: "kind-node",
		GoCmd:    "go",
		Arch:     "amd64",
	}
}

// Build builds the cluster node image, the sourcedir must be set on
// the NodeImageBuildContext
func (c *BaseImageBuildContext) Build() (err error) {
	// create tempdir to build in
	tmpDir, err := ioutil.TempDir("", "kind-build")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// populate with image sources
	// if SourceDir is unset, use the baked in sources
	buildDir := tmpDir
	if c.SourceDir == "" {
		// populate with image sources
		err = sources.RestoreAssets(buildDir, "images/base")
		if err != nil {
			return err
		}
		buildDir = filepath.Join(buildDir, "images", "base")

	} else {
		err = copyDir(c.SourceDir, buildDir)
		if err != nil {
			glog.Errorf("failed to copy sources to build dir %v", err)
			return err
		}
	}

	glog.Infof("Building node image in: %s", buildDir)

	// build the entrypoint binary first
	if err := c.buildEntrypoint(buildDir); err != nil {
		return err
	}

	// then the actual docker image
	return c.buildImage(buildDir)
}

// builds the entrypoint binary
func (c *BaseImageBuildContext) buildEntrypoint(dir string) error {
	// NOTE: this binary only uses the go1 stdlib, and is a single file
	entrypointSrc := filepath.Join(dir, "entrypoint", "main.go")
	entrypointDest := filepath.Join(dir, "entrypoint", "entrypoint")

	cmd := exec.Command(c.GoCmd, "build", "-o", entrypointDest, entrypointSrc)
	// TODO(bentheelder): we may need to map between docker image arch and GOARCH
	cmd.Env = []string{"GOOS=linux", "GOARCH=" + c.Arch}

	// actually build
	glog.Info("Building entrypoint binary ...")
	cmd.Debug = true
	cmd.InheritOutput = true
	if err := cmd.Run(); err != nil {
		glog.Errorf("Entrypoint build Failed! %v", err)
		return err
	}
	glog.Info("Entrypoint build completed.")
	return nil
}

func (c *BaseImageBuildContext) buildImage(dir string) error {
	// build the image, tagged as tagImageAs, using the our tempdir as the context
	cmd := exec.Command("docker", "build", "-t", c.ImageTag, dir)
	cmd.Debug = true
	cmd.InheritOutput = true

	glog.Info("Starting Docker build ...")
	err := cmd.Run()
	if err != nil {
		glog.Errorf("Docker build Failed! %v", err)
		return err
	}
	glog.Info("Docker build completed.")
	return nil
}

// TODO(bentheelder): vendor a portable go library for this and use instead
func copyDir(src, dst string) error {
	src = filepath.Clean(src) + string(filepath.Separator) + "."
	dst = filepath.Clean(dst)
	cmd := exec.Command("cp", "-r", src, dst)
	cmd.Debug = true
	cmd.InheritOutput = true
	return cmd.Run()
}
