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

package docker

import (
	"strings"

	"sigs.k8s.io/kind/pkg/exec"
)

// IsAvailable checks if docker is available in the system
func IsAvailable() bool {
	cmd := exec.Command("docker", "-v")
	lines, err := exec.OutputLines(cmd)
	if err != nil || len(lines) != 1 {
		return false
	}
	return strings.HasPrefix(lines[0], "Docker version")
}

// usernsRemap checks if userns-remap is enabled in dockerd
func usernsRemap() bool {
	cmd := exec.Command("docker", "info", "--format", "'{{json .SecurityOptions}}'")
	lines, err := exec.OutputLines(cmd)
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

// mountDevMapper checks if the Docker storage driver is Btrfs or ZFS
func mountDevMapper() bool {
	storage := ""
	cmd := exec.Command("docker", "info", "-f", "{{.Driver}}")
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return false
	}
	if len(lines) > 0 {
		storage = strings.ToLower(strings.TrimSpace(lines[0]))
	}
	return storage == "btrfs" || storage == "zfs"
}
