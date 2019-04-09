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
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/constants"

	"sigs.k8s.io/kind/pkg/exec"
)

// Delete deletes nodes by name / ID (see Node.String())
func Delete(nodes ...Node) error {
	if len(nodes) == 0 {
		return nil
	}
	ids := []string{}
	for _, node := range nodes {
		ids = append(ids, node.name)
	}
	cmd := exec.Command(
		"docker",
		append(
			[]string{
				"rm",
				"-f", // force the container to be delete now
				"-v", // delete volumes
			},
			ids...,
		)...,
	)
	return cmd.Run()
}

// List returns the list of container IDs for the kind "nodes", optionally
// filtered by docker ps filters
// https://docs.docker.com/engine/reference/commandline/ps/#filtering
func List(filters ...string) ([]Node, error) {
	res := []Node{}
	visit := func(cluster string, node *Node) {
		res = append(res, *node)
	}
	return res, list(visit, filters...)
}

// ListByCluster returns a list of nodes by the kind cluster name
func ListByCluster(filters ...string) (map[string][]Node, error) {
	res := make(map[string][]Node)
	visit := func(cluster string, node *Node) {
		res[cluster] = append(res[cluster], *node)
	}
	return res, list(visit, filters...)
}

func list(visit func(string, *Node), filters ...string) error {
	args := []string{
		"ps",
		"-q",         // quiet output for parsing
		"-a",         // show stopped nodes
		"--no-trunc", // don't truncate
		// filter for nodes with the cluster label
		"--filter", "label=" + constants.ClusterLabelKey,
		// format to include friendly name and the cluster name
		"--format", fmt.Sprintf(`{{.Names}}\t{{.Label "%s"}}`, constants.ClusterLabelKey),
	}
	for _, filter := range filters {
		args = append(args, "--filter", filter)
	}
	cmd := exec.Command("docker", args...)
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to list nodes")
	}
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return errors.Errorf("invalid output when listing nodes: %s", line)
		}
		names := strings.Split(parts[0], ",")
		cluster := parts[1]
		visit(cluster, FromName(names[0]))
	}
	return nil
}

// WaitForReady uses kubectl inside the "node" container to check if the
// control plane nodes are "Ready".
func WaitForReady(node *Node, until time.Time) bool {
	return tryUntil(until, func() bool {
		cmd := node.Command(
			"kubectl",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"get",
			"nodes",
			"--selector=node-role.kubernetes.io/master",
			// When the node reaches status ready, the status field will be set
			// to true.
			"-o=jsonpath='{.items..status.conditions[-1:].status}'",
		)
		lines, err := exec.CombinedOutputLines(cmd)
		if err != nil {
			return false
		}

		// 'lines' will return the status of all nodes labeled as master. For
		// example, if we have three control plane nodes, and all are ready,
		// then the status will have the following format: `True True True'.
		status := strings.Fields(lines[0])
		for _, s := range status {
			// Check node status. If node is ready then this wil be 'True',
			// 'False' or 'Unkown' otherwise.
			if !strings.Contains(s, "True") {
				return false
			}
		}
		return true
	})
}

// helper that calls `try()`` in a loop until the deadline `until`
// has passed or `try()`returns true, returns wether try ever returned true
func tryUntil(until time.Time, try func() bool) bool {
	for until.After(time.Now()) {
		if try() {
			return true
		}
	}
	return false
}
