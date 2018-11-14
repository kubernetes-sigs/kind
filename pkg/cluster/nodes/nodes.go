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

package nodes

import (
	"sigs.k8s.io/kind/pkg/cluster/consts"

	"sigs.k8s.io/kind/pkg/exec"
)

// Delete deletes nodes by name / ID (see Node.String())
func Delete(names ...string) error {
	cmd := exec.Command(
		"docker",
		append(
			[]string{
				"rm",
				"-f", // force the container to be delete now
				"-v", // delete volumes
			},
			names...,
		)...,
	)
	return cmd.Run()
}

// List returns the list of container IDs for the kind "nodes", optionally
// filtered by docker ps filters
// https://docs.docker.com/engine/reference/commandline/ps/#filtering
func List(filters ...string) (containerIDs []string, err error) {
	args := []string{
		"ps",
		"-q", // quiet output for parsing
		"-a", // show stopped nodes
		// filter for nodes with the cluster label
		"--filter", "label=" + consts.ClusterLabelKey,
	}
	for _, filter := range filters {
		args = append(args, "--filter", filter)
	}
	cmd := exec.Command("docker", args...)
	return exec.CombinedOutputLines(cmd)
}
