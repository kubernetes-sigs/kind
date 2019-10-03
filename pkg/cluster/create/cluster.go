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

package create

import (
	"time"

	"sigs.k8s.io/kind/pkg/apis/config/v1alpha3"
	internalencoding "sigs.k8s.io/kind/pkg/internal/apis/config/encoding"
	internaltypes "sigs.k8s.io/kind/pkg/internal/cluster/create/types"
)

// ClusterOption is a cluster creation option
type ClusterOption func(*internaltypes.ClusterOptions) (*internaltypes.ClusterOptions, error)

// WithConfigFile configures creating the cluster using the config file at path
func WithConfigFile(path string) ClusterOption {
	return func(o *internaltypes.ClusterOptions) (*internaltypes.ClusterOptions, error) {
		var err error
		o.Config, err = internalencoding.Load(path)
		return o, err
	}
}

// WithV1Alpha3 configures creating the cluster with a v1alpha3 config
func WithV1Alpha3(cluster *v1alpha3.Cluster) ClusterOption {
	return func(o *internaltypes.ClusterOptions) (*internaltypes.ClusterOptions, error) {
		o.Config = internalencoding.V1Alpha3ToInternal(cluster)
		return o, nil
	}
}

// WithNodeImage overrides the image on all nodes in config as an easy way
// to change the Kubernetes version
func WithNodeImage(nodeImage string) ClusterOption {
	return func(o *internaltypes.ClusterOptions) (*internaltypes.ClusterOptions, error) {
		o.NodeImage = nodeImage
		return o, nil
	}
}

// Retain configures create to retain nodes after failing for debugging pourposes
func Retain(retain bool) ClusterOption {
	return func(o *internaltypes.ClusterOptions) (*internaltypes.ClusterOptions, error) {
		o.Retain = retain
		return o, nil
	}
}

// WaitForReady configures create to use interval as maximum wait time for the control plane node to be ready
func WaitForReady(interval time.Duration) ClusterOption {
	return func(o *internaltypes.ClusterOptions) (*internaltypes.ClusterOptions, error) {
		o.WaitForReady = interval
		return o, nil
	}
}

// SetupKubernetes configures create command to setup kubernetes after creating nodes containers
// TODO: Refactor this. It is a temporary solution for a phased breakdown of different
//      operations, specifically create. see https://github.com/kubernetes-sigs/kind/issues/324
func SetupKubernetes(setupKubernetes bool) ClusterOption {
	return func(o *internaltypes.ClusterOptions) (*internaltypes.ClusterOptions, error) {
		o.SetupKubernetes = setupKubernetes
		return o, nil
	}
}
