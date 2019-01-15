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

package meta

import (
	"fmt"
	"path/filepath"

	"k8s.io/client-go/util/homedir"

	"sigs.k8s.io/kind/pkg/cluster/consts"
)

// ClusterMeta contains some cluster meta and can be used to compute some more
// cluster metadata
// See: NewClusterMeta
type ClusterMeta struct {
	name string
}

// NewClusterMeta returns a new cluster meta
func NewClusterMeta(name string) *ClusterMeta {
	return &ClusterMeta{
		name: name,
	}
}

// Name returns the cluster's name
func (c *ClusterMeta) Name() string {
	return c.name
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *ClusterMeta) KubeConfigPath() string {
	// configDir matches the standard directory expected by kubectl etc
	configDir := filepath.Join(homedir.HomeDir(), ".kube")
	// note that the file name however does not, we do not want to overwrite
	// the standard config, though in the future we may (?) merge them
	fileName := fmt.Sprintf("kind-config-%s", c.name)
	return filepath.Join(configDir, fileName)
}

// ClusterLabel returns the docker object label that will be applied
// to cluster "node" containers
func (c *ClusterMeta) ClusterLabel() string {
	return fmt.Sprintf("%s=%s", consts.ClusterLabelKey, c.name)
}
