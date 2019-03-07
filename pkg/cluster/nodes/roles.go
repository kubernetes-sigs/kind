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

package nodes

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/constants"
)

// SelectNodesByRole returns a list of nodes with the matching role
func SelectNodesByRole(allNodes []Node, role string) ([]Node, error) {
	out := []Node{}
	for _, node := range allNodes {
		nodeRole, err := node.Role()
		if err != nil {
			return nil, err
		}
		if nodeRole == role {
			out = append(out, node)
		}
	}
	return out, nil
}

// ExternalLoadBalancerNode returns a node handle for the external control plane
// loadbalancer node or nil if there isn't one
func ExternalLoadBalancerNode(allNodes []Node) (*Node, error) {
	// identify and validate external load balancer node
	loadBalancerNodes, err := SelectNodesByRole(
		allNodes,
		constants.ExternalLoadBalancerNodeRoleValue,
	)
	if err != nil {
		return nil, err
	}
	if len(loadBalancerNodes) < 1 {
		return nil, nil
	}
	if len(loadBalancerNodes) > 1 {
		return nil, errors.Errorf(
			"unexpected number of %s nodes %d",
			constants.ExternalLoadBalancerNodeRoleValue,
			len(loadBalancerNodes),
		)
	}
	return &loadBalancerNodes[0], nil
}

// ControlPlaneNodes returns all control plane nodes such that the first entry
// is the bootstrap control plane node
func ControlPlaneNodes(allNodes []Node) ([]Node, error) {
	controlPlaneNodes, err := SelectNodesByRole(
		allNodes,
		constants.ControlPlaneNodeRoleValue,
	)
	if err != nil {
		return nil, err
	}
	// pick the first by sorting
	// TODO(bentheelder): perhaps in the future we should mark this node
	// specially at container creation time
	sort.Slice(controlPlaneNodes, func(i, j int) bool {
		return strings.Compare(controlPlaneNodes[i].Name(), controlPlaneNodes[j].Name()) < 0
	})
	return controlPlaneNodes, nil
}

// BootstrapControlPlaneNode returns a handle to the bootstrap control plane node
func BootstrapControlPlaneNode(allNodes []Node) (*Node, error) {
	controlPlaneNodes, err := ControlPlaneNodes(allNodes)
	if err != nil {
		return nil, err
	}
	if len(controlPlaneNodes) < 1 {
		return nil, errors.Errorf(
			"expected at least one %s node",
			constants.ControlPlaneNodeRoleValue,
		)
	}
	return &controlPlaneNodes[0], nil
}

// SecondaryControlPlaneNodes returns handles to the secondary
// control plane nodes and NOT the bootstrap control plane node
func SecondaryControlPlaneNodes(allNodes []Node) ([]Node, error) {
	controlPlaneNodes, err := ControlPlaneNodes(allNodes)
	if err != nil {
		return nil, err
	}
	if len(controlPlaneNodes) < 1 {
		return nil, errors.Errorf(
			"expected at least one %s node",
			constants.ControlPlaneNodeRoleValue,
		)
	}
	return controlPlaneNodes[1:], nil
}
