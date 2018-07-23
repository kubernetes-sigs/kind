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
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
)

// NodeImageBuildContext is used to build the kind node image, and contains
// build configuration
type NodeImageBuildContext struct {
	SourceDir string
	ImageTag  string
	GoCmd     string
	Arch      string
}

// NewNodeImageBuildContext creates a new NodeImageBuildContext with
// default configuration
func NewNodeImageBuildContext() *NodeImageBuildContext {
	return &NodeImageBuildContext{
		ImageTag: "kind-node",
		GoCmd:    "go",
		Arch:     "amd64",
	}
}

func runCmd(cmd *exec.Cmd) error {
	glog.Infof("Running: %v %v", cmd.Path, cmd.Args)
	// TODO(bentheelder): reconsider this / make it configurable
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// TODO(bentheelder): vendor a portable go library for this and use instead
func copyDir(src, dst string) error {
	src = filepath.Clean(src) + string(filepath.Separator) + "."
	dst = filepath.Clean(dst)
	cmd := exec.Command("cp", "-r", src, dst)
	return runCmd(cmd)
}

// Build builds the cluster node image, the sourcedir must be set on
// the NodeImageBuildContext
func (c *NodeImageBuildContext) Build() (err error) {
	// create tempdir to build in
	dir, err := ioutil.TempDir("", "kind-build")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	// populate with image sources
	err = copyDir(c.SourceDir, dir)
	if err != nil {
		glog.Errorf("failed to copy sources to build dir %v", err)
		return err
	}

	glog.Infof("Building node image in: %s", dir)

	// build entrypoint binary
	// NOTE: this binary only uses the go1 stdlib, and is a single file
	glog.Info("Building entrypoint binary ...")
	entrypointSrc := filepath.Join(dir, "entrypoint", "main.go")
	entrypointDest := filepath.Join(dir, "entrypoint", "entrypoint")
	cmd := exec.Command(c.GoCmd, "build", "-o", entrypointDest, entrypointSrc)
	// TODO(bentheelder): we may need to map between docker image arch and GOARCH
	cmd.Env = []string{"GOOS=linux", "GOARCH=" + c.Arch}
	err = runCmd(cmd)
	if err != nil {
		glog.Errorf("Entrypoint build Failed! %v", err)
		return err
	}
	glog.Info("Entrypoint build completed.")

	glog.Info("Starting Docker build ...")
	// build the image, tagged as tagImageAs, using the our tempdir as the context
	err = runCmd(exec.Command("docker", "build", "-t", c.ImageTag, dir))
	if err != nil {
		glog.Errorf("Docker build Failed! %v", err)
		return err
	}
	glog.Info("Docker build completed.")
	return nil
}
