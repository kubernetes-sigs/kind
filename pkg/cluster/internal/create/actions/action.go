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

package actions

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Action defines a step of bringing up a kind cluster after initial contianer
// creation
type Action interface {
	Execute(ctx *ActionContext) error
}

// ActionContext is data supplied to all actions
type ActionContext struct {
	name   string
	Status *logutil.Status
	Nodes  []nodes.Node
	Config *config.Config
}

// NewActionContext returns a new ActionContext
func NewActionContext(
	status *logutil.Status, clusterNodes []nodes.Node, cfg *config.Config,
	clusterName string,
) *ActionContext {
	return &ActionContext{
		Status: status,
		Nodes:  clusterNodes,
		Config: cfg,
		name:   clusterName,
	}
}

// Name returns the cluster name
func (ac *ActionContext) Name() string {
	return ac.name
}

// SelectNodesByRole returns a list of nodes with the matching role
func (ac *ActionContext) SelectNodesByRole(role string) ([]*nodes.Node, error) {
	out := []*nodes.Node{}
	for _, node := range ac.Nodes {
		r, err := node.Role()
		if err != nil {
			return nil, err
		}
		if r == role {
			out = append(out, &node)
		}
	}
	return out, nil
}

// ExternalLoadBalancerNode returns a node handle for the external control plane
// loadbalancer node or nil if there isn't one
func (ac *ActionContext) ExternalLoadBalancerNode() (*nodes.Node, error) {
	// identify and validate external load balancer node
	loadBalancerNodes, err := ac.SelectNodesByRole(constants.ExternalLoadBalancerNodeRoleValue)
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
	return loadBalancerNodes[0], nil
}

// BootstrapControlPlaneNode returns a handle to the bootstrap control plane node
func (ac *ActionContext) BootstrapControlPlaneNode() (*nodes.Node, error) {
	controlPlaneNodes, err := ac.SelectNodesByRole(constants.ControlPlaneNodeRoleValue)
	if err != nil {
		return nil, err
	}
	if len(controlPlaneNodes) < 1 {
		return nil, errors.Errorf(
			"expected at least one %s node",
			constants.ExternalLoadBalancerNodeRoleValue,
		)
	}
	// pick the first by sorting
	// TODO(bentheelder): perhaps in the future we should mark this node
	// specially at container creation time
	sort.Slice(controlPlaneNodes, func(i, j int) bool {
		return strings.Compare(controlPlaneNodes[i].Name(), controlPlaneNodes[j].Name()) > 0
	})
	return controlPlaneNodes[0], nil
}
