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
)

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
// TODO(bentheelder): this should take options
func NewContext(name string) *Context {
	// wrap a new internal context
	return &Context{
		ic: internalcontext.NewContext(name),
	}
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Context) KubeConfigPath() string {
	return c.ic.KubeConfigPath()
}

// KubeConfig returns the KUBECONFIG for the cluster
// If internal is true, this will contain the internal IP etc.
// If internal is fale, this will contain the host IP etc.
func (c *Context) KubeConfig(internal bool) (string, error) {
	// TODO(bentheelder): move implementation to node provider
	n, err := c.ic.ListNodes()
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
	f, err := os.Open(c.KubeConfigPath())
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

// Create provisions and starts a kubernetes-in-docker cluster
func (c *Context) Create(options ...create.ClusterOption) error {
	return internalcreate.Cluster(c.ic, options...)
}

// Delete tears down a kubernetes-in-docker cluster
func (c *Context) Delete() error {
	return internaldelete.Cluster(c.ic)
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
// TODO: move to public nodes type
func (c *Context) ListNodes() ([]nodes.Node, error) {
	return c.ic.ListNodes()
}

// ListInternalNodes returns the list of container IDs for the "nodes" in the cluster
// that are not external
// TODO: move to public nodes type
func (c *Context) ListInternalNodes() ([]nodes.Node, error) {
	return c.ic.ListInternalNodes()
}

// CollectLogs will populate dir with cluster logs and other debug files
func (c *Context) CollectLogs(dir string) error {
	// TODO: should use ListNodes and Collect should handle nodes differently
	// based on role ...
	n, err := c.ListInternalNodes()
	if err != nil {
		return err
	}
	return internallogs.Collect(n, dir)
}
