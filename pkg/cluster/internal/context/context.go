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

// Package context contains the internal cluster context shared by various
// packages that implement the user face pkg/cluster.Context
package context

import (
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/docker"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/provider"
)

// Context is the private shared context underlying pkg/cluster.Context
//
// NOTE: this is the internal one, it should contain reasonably trivial
// methods that are safe to share between various user facing methods
// pkg/cluster.Context is a superset of this, packages like create and delete
// consume this
type Context struct {
	name string
	// cluster backend (docker, ...)
	provider provider.Provider
}

// NewContext returns a new internal cluster management context
// if name is "" the default name will be used
func NewContext(logger log.Logger, name string) *Context {
	if name == "" {
		name = constants.DefaultClusterName
	}
	return &Context{
		name:     name,
		provider: docker.NewProvider(logger),
	}
}

// NewProviderContext returns a new context with given provider and name
func NewProviderContext(p provider.Provider, name string) *Context {
	return &Context{
		name:     name,
		provider: p,
	}
}

// Name returns the cluster's name
func (c *Context) Name() string {
	return c.name
}

// Provider returns the provider of the context
func (c *Context) Provider() provider.Provider {
	return c.provider
}

// GetAPIServerEndpoint returns the cluster's API Server endpoint
func (c *Context) GetAPIServerEndpoint() (string, error) {
	return c.provider.GetAPIServerEndpoint(c.Name())
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
func (c *Context) ListNodes() ([]nodes.Node, error) {
	return c.provider.ListNodes(c.name)
}

// ListInternalNodes returns the list of container IDs for the "nodes" in the cluster
// that are not external
func (c *Context) ListInternalNodes() ([]nodes.Node, error) {
	clusterNodes, err := c.ListNodes()
	if err != nil {
		return nil, err
	}
	selectedNodes := []nodes.Node{}
	for _, node := range clusterNodes {
		nodeRole, err := node.Role()
		if err != nil {
			return nil, err
		}
		if nodeRole == constants.WorkerNodeRoleValue || nodeRole == constants.ControlPlaneNodeRoleValue {
			selectedNodes = append(selectedNodes, node)
		}
	}
	return selectedNodes, nil
}
