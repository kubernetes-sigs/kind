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

// Package node implements node management operations
package node

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/log"
)

// JoinNodeToCluster joins a new node to an existing cluster
func JoinNodeToCluster(logger log.Logger, provider providers.Provider, cluster string, node nodes.Node, nodeRole config.NodeRole) error {
	// Get all nodes in the cluster to find a control plane node
	allNodes, err := provider.ListNodes(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to list cluster nodes")
	}

	// Filter out the node we're trying to add to avoid selecting it as the bootstrap node
	var existingNodes []nodes.Node
	for _, n := range allNodes {
		if n.String() != node.String() {
			existingNodes = append(existingNodes, n)
		}
	}

	// Find a control plane node to get join information from
	controlPlaneNode, err := nodeutils.BootstrapControlPlaneNode(existingNodes)
	if err != nil {
		return errors.Wrap(err, "failed to find bootstrap control plane node")
	}

	// Generate join command from existing control plane node
	joinCommand, err := generateJoinCommand(logger, controlPlaneNode, nodeRole)
	if err != nil {
		return errors.Wrap(err, "failed to generate join command")
	}

	// Execute the join command directly on the new node
	if err := runJoinCommand(logger, node, joinCommand); err != nil {
		return errors.Wrap(err, "failed to run join command")
	}

	// Wait for the node to be ready
	if err := waitForNodeReady(logger, node, 2*time.Minute); err != nil {
		return errors.Wrap(err, "node failed to become ready")
	}

	// Wait for the node to be visible in kubectl (via API server)
	if err := waitForNodeVisibleInAPI(logger, controlPlaneNode, node.String(), 1*time.Minute); err != nil {
		return errors.Wrap(err, "node failed to become visible in Kubernetes API")
	}

	return nil
}

// generateJoinCommand creates a join command from an existing control plane node
func generateJoinCommand(logger log.Logger, controlPlaneNode nodes.Node, nodeRole config.NodeRole) ([]string, error) {
	// For control plane nodes, generate certificate key and use it in the join command
	if nodeRole == config.ControlPlaneRole {
		// Generate certificate key for control plane join
		certKey, err := generateCertificateKey(logger, controlPlaneNode)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate certificate key")
		}

		// Create join command with the certificate key
		cmd := controlPlaneNode.Command("kubeadm", "token", "create", "--print-join-command", "--certificate-key", certKey)
		lines, err := exec.CombinedOutputLines(cmd)
		if err != nil {
			logger.V(3).Info(strings.Join(lines, "\n"))
			return nil, errors.Wrap(err, "failed to create join token with certificate key")
		}

		joinCommand := strings.Join(lines, " ")
		return strings.Fields(joinCommand), nil
	}

	// For worker nodes, get the base join command
	cmd := controlPlaneNode.Command("kubeadm", "token", "create", "--print-join-command")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		logger.V(3).Info(strings.Join(lines, "\n"))
		return nil, errors.Wrap(err, "failed to create join token")
	}

	joinCommand := strings.Join(lines, " ")
	return strings.Fields(joinCommand), nil
}

// generateCertificateKey generates a random certificate key and uploads certificates
func generateCertificateKey(logger log.Logger, controlPlaneNode nodes.Node) (string, error) {
	// Generate a random 32-byte (256-bit) AES key as hex string
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", errors.Wrap(err, "failed to generate random certificate key")
	}
	certKey := hex.EncodeToString(keyBytes)

	// Upload certificates using our generated key
	cmd := controlPlaneNode.Command("kubeadm", "init", "phase", "upload-certs", "--certificate-key", certKey, "--upload-certs")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		logger.V(3).Info(strings.Join(lines, "\n"))
		return "", errors.Wrap(err, "failed to upload certificates")
	}

	logger.V(2).Infof("Successfully uploaded certificates with key: %s", certKey[:8]+"...")
	return certKey, nil
}

// runJoinCommand executes the kubeadm join command on the target node
func runJoinCommand(logger log.Logger, node nodes.Node, joinArgs []string) error {
	logger.V(0).Infof("Running kubeadm join on node %s", node.String())

	cmd := node.Command(joinArgs[0], joinArgs[1:]...)
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		logger.V(3).Info(strings.Join(lines, "\n"))
		return errors.Wrap(err, "kubeadm join failed")
	}

	return nil
}

// waitForNodeReady waits for the node to become ready
func waitForNodeReady(logger log.Logger, node nodes.Node, timeout time.Duration) error {
	logger.V(0).Infof("Waiting for node %s to be ready...", node.String())

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Check if kubelet is running
		cmd := node.Command("systemctl", "is-active", "kubelet")
		if err := cmd.Run(); err == nil {
			logger.V(0).Infof("Node %s is ready", node.String())
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return errors.Errorf("node %s did not become ready within %v", node.String(), timeout)
}

// waitForNodeVisibleInAPI waits for the node to be visible via kubectl on the API server
func waitForNodeVisibleInAPI(logger log.Logger, controlPlaneNode nodes.Node, nodeName string, timeout time.Duration) error {
	logger.V(0).Infof("Waiting for node %s to be visible in Kubernetes API...", nodeName)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Try to get the node via kubectl
		cmd := controlPlaneNode.Command("kubectl", "get", "nodes", nodeName, "--no-headers")
		if err := cmd.Run(); err == nil {
			logger.V(0).Infof("Node %s is visible in Kubernetes API", nodeName)
			return nil
		}

		// Log progress every 10 seconds to show we're still working
		if int(time.Now().Unix())%10 == 0 {
			logger.V(1).Infof("Still waiting for node %s to appear in kubectl...", nodeName)
		}

		time.Sleep(2 * time.Second)
	}

	return errors.Errorf("node %s did not become visible in Kubernetes API within %v", nodeName, timeout)
}
