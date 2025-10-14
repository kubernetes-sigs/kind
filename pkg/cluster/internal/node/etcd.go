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
	"fmt"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// EtcdMember represents an etcd cluster member
type EtcdMember struct {
	ID        string
	Name      string
	PeerURL   string
	ClientURL string
	IsLearner bool
}

// RemoveEtcdMember removes a control plane node from the etcd cluster
func RemoveEtcdMember(logger log.Logger, allNodes []nodes.Node, nodeToRemove string) error {
	logger.V(1).Infof("Removing etcd member for node: %s", nodeToRemove)

	// Filter out the node being removed to find a healthy control plane node
	var healthyNodes []nodes.Node
	for _, n := range allNodes {
		if n.String() != nodeToRemove {
			healthyNodes = append(healthyNodes, n)
		}
	}

	// Find a healthy control plane node to execute etcd commands from
	controlPlaneNode, err := nodeutils.BootstrapControlPlaneNode(healthyNodes)
	if err != nil {
		return errors.Wrap(err, "failed to find healthy control plane node for etcd operations")
	}

	// Get current etcd members
	members, err := listEtcdMembers(logger, controlPlaneNode)
	if err != nil {
		return errors.Wrap(err, "failed to list etcd members")
	}

	logger.V(2).Infof("Found %d etcd members before removal", len(members))
	for _, member := range members {
		logger.V(3).Infof("Member: %s (ID: %s)", member.Name, member.ID)
	}

	// Find the member to remove
	var memberToRemove *EtcdMember
	for _, member := range members {
		if member.Name == nodeToRemove {
			memberToRemove = &member
			break
		}
	}

	if memberToRemove == nil {
		logger.Warnf("Node %s not found in etcd member list, skipping etcd member removal", nodeToRemove)
		return nil
	}

	// Check that we're not removing the last member (should be caught earlier, but double-check)
	if len(members) <= 1 {
		return errors.New("cannot remove the last etcd member")
	}

	// Remove the member from etcd cluster
	logger.V(0).Infof("Removing etcd member %s (ID: %s)", memberToRemove.Name, memberToRemove.ID)
	if err := removeEtcdMemberByID(logger, controlPlaneNode, memberToRemove.ID); err != nil {
		return errors.Wrapf(err, "failed to remove etcd member %s", memberToRemove.ID)
	}

	// Verify the member was removed
	remainingMembers, err := listEtcdMembers(logger, controlPlaneNode)
	if err != nil {
		logger.Warnf("Could not verify etcd member removal: %v", err)
		return nil // Don't fail the entire operation if verification fails
	}

	logger.V(0).Infof("Etcd cluster now has %d members after removing %s", len(remainingMembers), nodeToRemove)
	return nil
}

// executeEtcdCommand executes an etcdctl command inside the etcd container
func executeEtcdCommand(controlPlaneNode nodes.Node, args ...string) ([]string, error) {
	// First get the etcd container ID
	criCtlCmd := controlPlaneNode.Command("crictl", "ps", "--name", "etcd", "-q")
	containerLines, err := exec.OutputLines(criCtlCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find etcd container")
	}
	if len(containerLines) == 0 {
		return nil, errors.New("no etcd container found")
	}
	etcdContainerID := strings.TrimSpace(containerLines[0])

	// Build etcdctl command with TLS certificates and provided arguments
	etcdCtlArgs := []string{
		"crictl", "exec", etcdContainerID, "etcdctl",
		fmt.Sprintf("--endpoints=https://127.0.0.1:%d", common.EtcdClientPort),
		"--cacert=/etc/kubernetes/pki/etcd/ca.crt",
		"--cert=/etc/kubernetes/pki/etcd/server.crt",
		"--key=/etc/kubernetes/pki/etcd/server.key",
	}
	etcdCtlArgs = append(etcdCtlArgs, args...)

	etcdCtlCmd := controlPlaneNode.Command(etcdCtlArgs[0], etcdCtlArgs[1:]...)
	return exec.CombinedOutputLines(etcdCtlCmd)
}

// listEtcdMembers retrieves the list of etcd cluster members
func listEtcdMembers(logger log.Logger, controlPlaneNode nodes.Node) ([]EtcdMember, error) {
	lines, err := executeEtcdCommand(controlPlaneNode, "member", "list")
	if err != nil {
		logger.V(3).Infof("etcdctl member list output: %v", lines)
		return nil, errors.Wrap(err, "failed to list etcd members")
	}

	var members []EtcdMember
	for _, line := range lines {
		member, err := parseEtcdMemberLine(line)
		if err != nil {
			logger.V(2).Infof("Skipping unparseable etcd member line: %s", line)
			continue
		}
		members = append(members, member)
	}

	return members, nil
}

// removeEtcdMemberByID removes an etcd member by ID
func removeEtcdMemberByID(logger log.Logger, controlPlaneNode nodes.Node, memberID string) error {
	lines, err := executeEtcdCommand(controlPlaneNode, "member", "remove", memberID)
	if err != nil {
		logger.V(3).Infof("etcdctl member remove output: %v", lines)
		return errors.Wrapf(err, "failed to remove etcd member %s", memberID)
	}

	logger.V(2).Infof("Successfully removed etcd member %s", memberID)
	return nil
}

// parseEtcdMemberLine parses a line from etcdctl member list output
// Format: memberID, started, memberName, peerURL, clientURL, isLearner
// Example: e58c878e0e01014, started, etcd-test-control-plane, https://172.18.0.2:2380, https://172.18.0.2:2379, false
func parseEtcdMemberLine(line string) (EtcdMember, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return EtcdMember{}, errors.New("empty line")
	}

	parts := strings.Split(line, ",")
	if len(parts) < 6 {
		return EtcdMember{}, errors.Errorf("invalid etcd member line format: %s", line)
	}

	member := EtcdMember{
		ID:        strings.TrimSpace(parts[0]),
		Name:      strings.TrimSpace(parts[2]),
		PeerURL:   strings.TrimSpace(parts[3]),
		ClientURL: strings.TrimSpace(parts[4]),
		IsLearner: strings.TrimSpace(parts[5]) == "true",
	}

	return member, nil
}
