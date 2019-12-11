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
	"sort"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/log"

	internalcontext "sigs.k8s.io/kind/pkg/cluster/internal/context"
	internalcreate "sigs.k8s.io/kind/pkg/cluster/internal/create"
	internaldelete "sigs.k8s.io/kind/pkg/cluster/internal/delete"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeconfig"
	internallogs "sigs.k8s.io/kind/pkg/cluster/internal/logs"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/docker"
	internalprovider "sigs.k8s.io/kind/pkg/cluster/internal/providers/provider"
)

// DefaultName is the default cluster name
const DefaultName = constants.DefaultClusterName

// Provider is used to perform cluster operations
type Provider struct {
	provider internalprovider.Provider
	logger   log.Logger
}

// NewProvider returns a new provider based on the supplied options
func NewProvider(options ...ProviderOption) *Provider {
	p := &Provider{
		logger: log.NoopLogger{},
	}
	// Ensure we apply the logger options first, while maintaining the order
	// otherwise. This way we can trivially init the internal provider with
	// the logger.
	sort.SliceStable(options, func(i, j int) bool {
		_, iIsLogger := options[i].(providerLoggerOption)
		_, jIsLogger := options[j].(providerLoggerOption)
		return iIsLogger && !jIsLogger
	})
	for _, o := range options {
		o.apply(p)
	}
	if p.provider == nil {
		p.provider = docker.NewProvider(p.logger)
	}
	return p
}

// ProviderOption is an option for configuring a provider
type ProviderOption interface {
	apply(p *Provider)
}

// providerLoggerOption is a trivial ProviderOption adapter
// we use a type specific to logging options so we can handle them first
type providerLoggerOption func(p *Provider)

func (a providerLoggerOption) apply(p *Provider) {
	a(p)
}

// ProviderWithLogger configures the provider to use Logger logger
func ProviderWithLogger(logger log.Logger) ProviderOption {
	return providerLoggerOption(func(p *Provider) {
		p.logger = logger
	})
}

// TODO: remove this, rename internal context to something else
func (p *Provider) ic(name string) *internalcontext.Context {
	return internalcontext.NewProviderContext(p.provider, name)
}

// Create provisions and starts a kubernetes-in-docker cluster
func (p *Provider) Create(name string, options ...CreateOption) error {
	// apply options
	opts := &internalcreate.ClusterOptions{}
	for _, o := range options {
		if err := o.apply(opts); err != nil {
			return err
		}
	}
	return internalcreate.Cluster(p.logger, p.ic(name), opts)
}

// Delete tears down a kubernetes-in-docker cluster
func (p *Provider) Delete(name, explicitKubeconfigPath string) error {
	return internaldelete.Cluster(p.logger, p.ic(name), explicitKubeconfigPath)
}

// List returns a list of clusters for which nodes exist
func (p *Provider) List() ([]string, error) {
	return p.provider.ListClusters()
}

// KubeConfig returns the KUBECONFIG for the cluster
// If internal is true, this will contain the internal IP etc.
// If internal is false, this will contain the host IP etc.
func (p *Provider) KubeConfig(name string, internal bool) (string, error) {
	return kubeconfig.Get(p.ic(name), !internal)
}

// ExportKubeConfig exports the KUBECONFIG for the cluster, merging
// it into the selected file, following the rules from
// https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#config
// where explicitPath is the --kubeconfig value.
func (p *Provider) ExportKubeConfig(name string, explicitPath string) error {
	return kubeconfig.Export(p.ic(name), explicitPath)
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
	return internallogs.Collect(p.logger, n, dir)
}
