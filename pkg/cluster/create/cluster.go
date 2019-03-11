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

	internalcreate "sigs.k8s.io/kind/pkg/cluster/internal/create"
)

// ClusterOption is a cluster creation option
type ClusterOption func(*internalcreate.Options) *internalcreate.Options

// Retain configures create to retain nodes after failing for debugging pourposes
func Retain(retain bool) ClusterOption {
	return func(o *internalcreate.Options) *internalcreate.Options {
		o.Retain = retain
		return o
	}
}

// WaitForReady configures create to use interval as maximum wait time for the control plane node to be ready
func WaitForReady(interval time.Duration) ClusterOption {
	return func(o *internalcreate.Options) *internalcreate.Options {
		o.WaitForReady = interval
		return o
	}
}

// SetupKubernetes configures create command to setup kubernetes after creating nodes containers
// TODO: Refactor this. It is a temporary solution for a phased breakdown of different
//      operations, specifically create. see https://github.com/kubernetes-sigs/kind/issues/324
func SetupKubernetes(setupKubernetes bool) ClusterOption {
	return func(o *internalcreate.Options) *internalcreate.Options {
		o.SetupKubernetes = setupKubernetes
		return o
	}
}
