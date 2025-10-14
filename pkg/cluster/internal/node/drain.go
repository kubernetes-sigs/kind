/*
Copyright 2025 The Kubernetes Authors.

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

package node

import (
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// DrainAndRemoveNode safely drains and removes a node from the Kubernetes cluster
func DrainAndRemoveNode(logger log.Logger, provider providers.Provider, cluster, nodeName string) error {
	// Get all nodes to find a control plane node for kubectl operations
	allNodes, err := provider.ListNodes(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to list cluster nodes")
	}

	// Find a control plane node to run kubectl from
	controlPlaneNode, err := nodeutils.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return errors.Wrap(err, "failed to find bootstrap control plane node")
	}

	// Cordon the node first
	logger.V(0).Infof("Cordoning node %s...", nodeName)
	if err := cordonNode(controlPlaneNode, nodeName); err != nil {
		logger.Warnf("Failed to cordon node %s: %v", nodeName, err)
		// Continue with drain even if cordon fails
	}

	// Drain the node
	logger.V(0).Infof("Draining node %s...", nodeName)
	if err := drainNode(logger, controlPlaneNode, nodeName); err != nil {
		logger.Warnf("Failed to drain node %s: %v", nodeName, err)
		// Continue with removal even if drain fails
	}

	// Remove the node from the cluster
	logger.V(0).Infof("Removing node %s from cluster...", nodeName)
	if err := deleteNode(controlPlaneNode, nodeName); err != nil {
		logger.Warnf("Failed to delete node %s from cluster: %v", nodeName, err)
		// Continue with cleanup even if delete fails
	}

	return nil
}

// cordonNode marks the node as unschedulable
func cordonNode(controlPlaneNode nodes.Node, nodeName string) error {
	cmd := controlPlaneNode.Command("kubectl", "cordon", nodeName)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to cordon node")
	}
	return nil
}

// drainNode safely evicts all pods from the node
func drainNode(logger log.Logger, controlPlaneNode nodes.Node, nodeName string) error {
	args := []string{
		"drain",
		nodeName,
		"--ignore-daemonsets",
		"--delete-emptydir-data",
		"--force",
		"--timeout=60s",
	}

	cmd := controlPlaneNode.Command("kubectl", args...)
	lines, err := exec.CombinedOutputLines(cmd)
	logger.V(3).Info(strings.Join(lines, "\n"))

	if err != nil {
		return errors.Wrap(err, "failed to drain node")
	}

	return nil
}

// deleteNode removes the node from the cluster
func deleteNode(controlPlaneNode nodes.Node, nodeName string) error {
	cmd := controlPlaneNode.Command("kubectl", "delete", "node", nodeName)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to delete node from cluster")
	}
	return nil
}

// ResetNode runs kubeadm reset on a node to clean up Kubernetes components
func ResetNode(logger log.Logger, node nodes.Node) error {
	logger.V(0).Infof("Resetting Kubernetes components on node %s...", node.String())

	args := []string{
		"reset",
		"--force",
	}

	cmd := node.Command("kubeadm", args...)
	lines, err := exec.CombinedOutputLines(cmd)
	logger.V(3).Info(strings.Join(lines, "\n"))

	if err != nil {
		logger.Warnf("Failed to reset node %s: %v", node.String(), err)
		// Don't fail the entire operation if reset fails
	}

	return nil
}
