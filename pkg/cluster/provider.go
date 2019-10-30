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
	"bytes"
	"io/ioutil"
	"os"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/create"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	internalcontext "sigs.k8s.io/kind/pkg/internal/cluster/context"
	internalcreate "sigs.k8s.io/kind/pkg/internal/cluster/create"
	internaldelete "sigs.k8s.io/kind/pkg/internal/cluster/delete"
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
func (p *Provider) Delete(name string) error {
	return internaldelete.Cluster(p.ic(name))
}

// List returns a list of clusters for which nodes exist
func (p *Provider) List() ([]string, error) {
	return p.provider.ListClusters()
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (p *Provider) KubeConfigPath(name string) string {
	return p.ic(name).KubeConfigPath()
}

// KubeConfig returns the KUBECONFIG for the cluster
// If internal is true, this will contain the internal IP etc.
// If internal is fale, this will contain the host IP etc.
func (p *Provider) KubeConfig(name string, internal bool) (string, error) {
	// TODO(bentheelder): move implementation to node provider
	n, err := p.ic(name).ListNodes()
	if err != nil {
		return "", err
	}
	if internal {
		var buff bytes.Buffer
		nodes, err := nodeutils.ControlPlaneNodes(n)
		if err != nil {
			return "", err
		}
		if len(nodes) < 1 {
			return "", errors.New("could not locate any control plane nodes")
		}
		node := nodes[0]
		// grab kubeconfig version from one of the control plane nodes
		if err := node.Command("cat", "/etc/kubernetes/admin.conf").SetStdout(&buff).Run(); err != nil {
			return "", errors.Wrap(err, "failed to get cluster internal kubeconfig")
		}
		return buff.String(), nil
	}

	// TODO(bentheelder): should not depend on host kubeconfig file!
	f, err := os.Open(p.KubeConfigPath(name))
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster kubeconfig")
	}
	defer f.Close()
	out, err := ioutil.ReadAll(f)
	if err != nil {
		return "", errors.Wrap(err, "failed to read kubeconfig")
	}
	return string(out), nil
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
