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

package kube

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"al.essio.dev/pkg/shellescape"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// FindSource attempts to locate a kubernetes checkout using go's build package
func FindSource() (root string, err error) {
	// check current working directory first
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to locate kubernetes source, could not current working directory: %w", err)
	}
	if probablyKubeDir(wd) {
		return wd, nil
	}
	// then look under GOPATH
	gopath := build.Default.GOPATH
	if gopath == "" {
		return "", errors.New("could not find Kubernetes source under current working directory and GOPATH is not set")
	}
	// try k8s.io/kubernetes first (old canonical GOPATH locaation)
	if dir := filepath.Join(gopath, "src", "k8s.io", "kubernetes"); probablyKubeDir(dir) {
		return dir, nil
	}
	// then try github.com/kubernetes/kubernetes (CI without path_alias set)
	if dir := filepath.Join(gopath, "src", "github.com", "kubernetes", "kubernetes"); probablyKubeDir(dir) {
		return dir, nil
	}
	return "", fmt.Errorf("could not find Kubernetes source under current working directory or GOPATH=%s", build.Default.GOPATH)
}

// probablyKubeDir returns true if the dir looks plausibly like a kubernetes
// source directory
func probablyKubeDir(dir string) bool {
	// TODO: should we do more checks?
	// NOTE: go.mod with this line has existed since Kubernetes 1.15
	const sentinelLine = "module k8s.io/kubernetes"
	contents, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(contents), sentinelLine)
}

// sourceVersion the kubernetes git version based on hack/print-workspace-status.sh
// the raw version is also returned
func sourceVersion(kubeRoot string) (string, error) {
	// get the version output
	cmd := exec.Command(
		"sh", "-c",
		fmt.Sprintf(
			"cd %s && hack/print-workspace-status.sh",
			shellescape.Quote(kubeRoot),
		),
	)
	output, err := exec.OutputLines(cmd)
	if err != nil {
		return "", err
	}

	// parse it, and populate it into _output/git_version
	version := ""
	for _, line := range output {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return "", errors.Errorf("could not parse kubernetes version: %q", strings.Join(output, "\n"))
		}
		if parts[0] == "gitVersion" {
			version = parts[1]
			return version, nil
		}
	}
	if version == "" {
		return "", errors.Errorf("could not obtain kubernetes version: %q", strings.Join(output, "\n"))

	}
	return "", errors.Errorf("could not find kubernetes version in output: %q", strings.Join(output, "\n"))
}
