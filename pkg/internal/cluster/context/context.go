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
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"

	"sigs.k8s.io/kind/pkg/internal/cluster/providers/docker"
	"sigs.k8s.io/kind/pkg/internal/cluster/providers/provider"
	"sigs.k8s.io/kind/pkg/internal/util/env"
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
func NewContext(name string) *Context {
	if name == "" {
		name = constants.DefaultClusterName
	}
	return &Context{
		name:     name,
		provider: docker.NewProvider(),
	}
}

// Name returns the cluster's name
func (c *Context) Name() string {
	return c.name
}

func (c *Context) Provider() provider.Provider {
	return c.provider
}

func (c *Context) GetAPIServerEndpoint() (string, error) {
	return c.provider.GetAPIServerEndpoint(c.Name())
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Context) KubeConfigPath() string {
	// configDir matches the standard directory expected by kubectl etc
	configDir := filepath.Join(env.HomeDir(), ".kube")
	// note that the file name however does not, we do not want to overwrite
	// the standard config, though in the future we may (?) merge them
	fileName := fmt.Sprintf("kind-config-%s", c.Name())
	return filepath.Join(configDir, fileName)
}

// ClusterLabel returns the docker object label that will be applied
// to cluster "node" containers
func (c *Context) ClusterLabel() string {
	return fmt.Sprintf("%s=%s", constants.ClusterLabelKey, c.Name())
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
		// Don't load image on external nodes like the load balancer
		nodeRole, err := node.Role()
		if err != nil {
			return nil, err
		}
		if nodeRole != constants.ExternalLoadBalancerNodeRoleValue {
			selectedNodes = append(selectedNodes, node)
		}
	}
	return selectedNodes, nil
}
