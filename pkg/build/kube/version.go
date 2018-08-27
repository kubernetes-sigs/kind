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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/test-infra/kind/pkg/exec"
)

// buildVersionFile creates a file for the kubernetes git version in
// ./_output/version based on hack/print-workspace-status.sh,
// these are built into the node image and consumed by the cluster tooling
func buildVersionFile(kubeRoot string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	os.Chdir(kubeRoot)
	// make sure we cd back when done
	defer os.Chdir(cwd)

	// get the version output
	cmd := exec.Command("hack/print-workspace-status.sh")
	cmd.Debug = true
	output, err := cmd.CombinedOutputLines()
	if err != nil {
		return err
	}
	outputDir := filepath.Join(kubeRoot, "_output")
	// parse it, and populate it into _output/git_version
	for _, line := range output {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return fmt.Errorf("could not parse kubernetes version")
		}
		if parts[0] == "gitVersion" {
			ioutil.WriteFile(
				filepath.Join(outputDir, "git_version"),
				[]byte(parts[1]),
				0777,
			)
		}
	}
	return nil
}
