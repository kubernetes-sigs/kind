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

// Package cluster implements kind kubernetes-in-docker cluster management
package cluster

import (
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/create"
	"sigs.k8s.io/kind/pkg/cluster/nodes"

	internalcontext "sigs.k8s.io/kind/pkg/internal/cluster/context"
	internalcreate "sigs.k8s.io/kind/pkg/internal/cluster/create"
	internaldelete "sigs.k8s.io/kind/pkg/internal/cluster/delete"
	"sigs.k8s.io/kind/pkg/internal/cluster/kubeconfig"
	internallogs "sigs.k8s.io/kind/pkg/internal/cluster/logs"
	"sigs.k8s.io/kind/pkg/internal/cluster/providers/docker"
	internalprovider "sigs.k8s.io/kind/pkg/internal/cluster/providers/provider"
)

// DefaultName is the default cluster name
const DefaultName = constants.DefaultClusterName

// Provider is used to perform cluster operations
type Provider struct {
	provider internalprovider.Provider
}

// NewProvider returns a new provider based on the supplied options
func NewProvider(options ...ProviderOption) *Provider {
	p := &Provider{}
	for _, o := range options {
		p = o(p)
	}
	if p.provider == nil {
		p.provider = docker.NewProvider()
	}
	return p
}

// ProviderOption is an option for configuring a provider
type ProviderOption func(*Provider) *Provider

// TODO: remove this, rename internal context to something else
func (p *Provider) ic(name string) *internalcontext.Context {
	return internalcontext.NewProviderContext(p.provider, name)
}

// Create provisions and starts a kubernetes-in-docker cluster
func (p *Provider) Create(name string, options ...create.ClusterOption) error {
	return internalcreate.Cluster(p.ic(name), options...)
}

// Delete tears down a kubernetes-in-docker cluster
func (p *Provider) Delete(name, explicitKubeconfigPath string) error {
	return internaldelete.Cluster(p.ic(name), explicitKubeconfigPath)
}

// List returns a list of clusters for which nodes exist
func (p *Provider) List() ([]string, error) {
	return p.provider.ListClusters()
}

// KubeConfig returns the KUBECONFIG for the cluster
// If internal is true, this will contain the internal IP etc.
// If internal is fale, this will contain the host IP etc.
func (p *Provider) KubeConfig(name string, internal bool) (string, error) {
	return kubeconfig.Get(p.ic(name), internal)
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
func (p *Provider) ListNodes(name string) ([]nodes.Node, error) {
	return p.ic(name).ListNodes()
}

// ListInternalNodes returns the list of container IDs for the "nodes" in the cluster
// that are not external
func (p *Provider) ListInternalNodes(name string) ([]nodes.Node, error) {
	return p.ic(name).ListInternalNodes()
}

// CollectLogs will populate dir with cluster logs and other debug files
func (p *Provider) CollectLogs(name, dir string) error {
	// TODO: should use ListNodes and Collect should handle nodes differently
	// based on role ...
	n, err := p.ListInternalNodes(name)
	if err != nil {
		return err
	}
	return internallogs.Collect(n, dir)
}
