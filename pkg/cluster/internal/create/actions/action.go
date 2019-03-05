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
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Action define a set of tasks to be executed on a `kind` cluster.
// TODO(bentheelder): redesign this for concurrency
// Usage of actions allows to define repetitive, high level abstractions/workflows
// by composing lower level tasks
type Action interface {
	Execute(ctx *ActionContext) error
}

// ActionContext is data supplied to all actions
type ActionContext struct {
	Status *logutil.Status
	Nodes  []nodes.Node
}

// NewActionContext returns a new ActionContext
func NewActionContext(status *logutil.Status, clusterNodes []nodes.Node) *ActionContext {
	return &ActionContext{
		Status: status,
		Nodes:  clusterNodes,
	}
}

// SelectNodesByRole returns a list of nodes with the matching role
func (ac *ActionContext) SelectNodesByRole(role string) ([]*nodes.Node, error) {
	out := []*nodes.Node{}
	for _, node := range ac.nodeByName {
		r, err := node.Role()
		if err != nil {
			return nil, err
		}
		if r == role {
			out = append(out, node)
		}
	}
	return out, nil
}

// ExternalLoadBalancerNode returns a node handle for the external control plane
// loadbalancer node or nil if there isn't one
func (ac *ActionContext) ExternalLoadBalancerNode() (*nodes.Node, error) {
	// identify and validate external load balancer node
	loadBalancerNodes, err := ctx.SelectNodesByRole(constants.ExternalLoadBalancerNodeRoleValue)
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
