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
	"strings"

	"github.com/alessio/shellescape"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// ImportPath is the canonical import path for the kubernetes root package
// this is used by FindSource
const ImportPath = "k8s.io/kubernetes"

// FindSource attempts to locate a kubernetes checkout using go's build package
func FindSource() (root string, err error) {
	// look up the source the way go build would
	pkg, err := build.Default.Import(ImportPath, build.Default.GOPATH, build.FindOnly|build.IgnoreVendor)
	if err == nil && maybeKubeDir(pkg.Dir) {
		return pkg.Dir, nil
	}
	return "", errors.New("could not find kubernetes source")
}

// maybeKubeDir returns true if the dir looks plausibly like a kubernetes
// source directory
func maybeKubeDir(dir string) bool {
	// TODO(bentheelder): consider adding other sanity checks
	return dir != ""
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
