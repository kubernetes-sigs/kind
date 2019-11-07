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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// buildVersionFile creates a file for the kubernetes git version in
// ./_output/version based on hack/print-workspace-status.sh,
// these are built into the node image and consumed by the cluster tooling
// the raw version is also returned
func buildVersionFile(logger log.Logger, kubeRoot string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	// make sure we cd back when done
	defer func() {
		// TODO(bentheelder): set return error?
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(kubeRoot); err != nil {
		return err
	}

	// get the version output
	cmd := exec.Command("hack/print-workspace-status.sh")
	output, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return err
	}

	// we will place the file in _output with other build artifacts
	outputDir := filepath.Join(kubeRoot, "_output")
	// ensure output dir, if we are using bazel it may not exist...
	// we can ignore the error because it either exists and we don't care
	// or if it fails to create the dir we'll see the file write error below
	// we do not use MkdirAll because kubeRoot better already exist..
	_ = os.Mkdir(outputDir, os.ModePerm)

	// parse it, and populate it into _output/git_version
	version := ""
	for _, line := range output {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			logger.Error("Could not parse kubernetes version, output: " + strings.Join(output, "\n"))
			return errors.New("could not parse kubernetes version")
		}
		if parts[0] == "gitVersion" {
			version = parts[1]
			if err := ioutil.WriteFile(
				filepath.Join(outputDir, "git_version"),
				[]byte(version),
				0777,
			); err != nil {
				return errors.Wrap(err, "failed to write version file")
			}
		}
	}
	if version == "" {
		logger.Error("Could not obtain kubernetes version, output: " + strings.Join(output, "\n"))
		return errors.New("could not obtain kubernetes version")
	}
	return nil
}
