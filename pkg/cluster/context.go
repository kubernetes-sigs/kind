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

package cluster

import (
	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/create"
	internalcontext "sigs.k8s.io/kind/pkg/cluster/internal/context"
	internalcreate "sigs.k8s.io/kind/pkg/cluster/internal/create"
	internaldelete "sigs.k8s.io/kind/pkg/cluster/internal/delete"
	"sigs.k8s.io/kind/pkg/cluster/logs"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

// TODO(bentheelder): reimplement GetControlPlaneMeta for Context

// DefaultName is the default cluster name
const DefaultName = constants.DefaultClusterName

// Context is used to create / manipulate kubernetes-in-docker clusters
// See: NewContext()
type Context struct {
	// the internal context type, shared between implementations of more
	// advanced methods like create
	ic *internalcontext.Context
}

// NewContext returns a new cluster management context
// if name is "" the default name will be used (constants.DefaultClusterName)
func NewContext(name string) *Context {
	// wrap a new internal context
	return &Context{
		ic: internalcontext.NewContext(name),
	}
}

// Validate will be called before creating new resources using the context
// It will not be called before deleting or listing resources, so as to allow
// contexts based around previously valid values to be used in newer versions
// You can call this early yourself to check validation before creation calls,
// though it will be called internally.
func (c *Context) Validate() error {
	return c.ic.Validate()
}

// Name returns context name / cluster name
func (c *Context) Name() string {
	return c.ic.Name()
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Context) KubeConfigPath() string {
	return c.ic.KubeConfigPath()
}

// Create provisions and starts a kubernetes-in-docker cluster
func (c *Context) Create(cfg *config.Cluster, options ...create.ClusterOption) error {
	// apply create options
	opts := &internalcreate.Options{
		SetupKubernetes: true,
	}
	for _, option := range options {
		opts = option(opts)
	}
	return internalcreate.Cluster(c.ic, cfg, opts)
}

// Delete tears down a kubernetes-in-docker cluster
func (c *Context) Delete() error {
	return internaldelete.Cluster(c.ic)
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
func (c *Context) ListNodes() ([]nodes.Node, error) {
	return c.ic.ListNodes()
}

// ListInternalNodes returns the list of container IDs for the "nodes" in the cluster
// that are not external
func (c *Context) ListInternalNodes() ([]nodes.Node, error) {
	return c.ic.ListInternalNodes()
}

// CollectLogs will populate dir with cluster logs and other debug files
func (c *Context) CollectLogs(dir string) error {
	nodes, err := c.ListInternalNodes()
	if err != nil {
		return err
	}
	return logs.Collect(nodes, dir)
}
