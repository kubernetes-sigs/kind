/*
Copyright 2026 The Kubernetes Authors.

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

package swarm

import (
	"strings"

	"sigs.k8s.io/kind/pkg/exec"
)

// IsAvailable checks if docker is available locally.  Per-host availability
// is verified at Provision time via `docker --context=<ctx> info`.
func IsAvailable() bool {
	cmd := exec.Command("docker", "-v")
	lines, err := exec.OutputLines(cmd)
	if err != nil || len(lines) != 1 {
		return false
	}
	return strings.HasPrefix(lines[0], "Docker version")
}

// usernsRemap checks if userns-remap is enabled in dockerd on the given host.
func usernsRemap(ctxName string) bool {
	cmd := exec.Command("docker",
		dockerArgs(ctxName, "info", "--format", "'{{json .SecurityOptions}}'")...,
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return false
	}
	if len(lines) > 0 && strings.Contains(lines[0], "name=userns") {
		return true
	}
	return false
}
