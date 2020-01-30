/*
Copyright 2019 The Kubernetes Authors.

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

package podman

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/version"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// usernsRemap checks if userns-remap is enabled in podmand
func usernsRemap() bool {
	cmd := exec.Command("podman", "info", "--format", "'{{json .SecurityOptions}}'")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return false
	}
	if len(lines) > 0 {
		if strings.Contains(lines[0], "name=userns") {
			return true
		}
	}
	return false
}

func getPodmanVersion() (*version.Version, error) {
	cmd := exec.Command("podman", "--version")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return nil, err
	}

	// output is like `podman version 1.7.1-dev`
	if len(lines) != 1 {
		return nil, errors.Errorf("podman version should only be one line, got %d", len(lines))
	}
	parts := strings.Split(lines[0], " ")
	if len(parts) != 3 {
		return nil, errors.Errorf("podman --version contents should have 3 parts, got %q", lines[0])
	}
	return version.ParseSemantic(parts[2])
}

const (
	// TODO: probably should be a released version ...
	minSupportedVersion = "1.7.1-dev"
)

func ensureMinVersion() error {
	// ensure that podman version is a compatible version
	v, err := getPodmanVersion()
	if err != nil {
		return errors.Wrap(err, "failed to check podman version")
	}
	if !v.AtLeast(version.MustParseSemantic(minSupportedVersion)) {
		return errors.Errorf("podman version %q is too old, please upgrade to %q or later", v, minSupportedVersion)
	}
	return nil
}
