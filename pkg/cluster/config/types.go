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

package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kind/pkg/kustomize"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config groups all nodes in the `kind` Config.
type Config struct {
	// TypeMeta representing the type of the object and its API schema version.
	metav1.TypeMeta

	// Nodes constains the list of nodes defined in the `kind` Config
	Nodes []Node `json:"nodes,"`
}

// Node contains settings for a node in the `kind` Config.
// A node in kind config represent a container that will be provisioned with all the components
// required for the assigned role in the Kubernetes cluster.
// If replicas is set, the desired node replica number will be generated.
type Node struct {
	// Replicas is the number of desired node replicas.
	// Defaults to 1
	Replicas *int32
	// Role defines the role of the nodw in the in the Kubernetes cluster managed by `kind`
	// Defaults to "control-plane"
	Role NodeRole
	// Image is the node image to use when running the cluster
	// TODO(bentheelder): split this into image and tag?
	Image string
	// KubeadmConfigPatches are applied to the generated kubeadm config as
	// strategic merge patches to `kustomize build` internally
	// https://github.com/kubernetes/community/blob/master/contributors/devel/strategic-merge-patch.md
	// This should be an inline yaml blob-string
	KubeadmConfigPatches []string
	// KubeadmConfigPatchesJSON6902 are applied to the generated kubeadm config
	// as patchesJson6902 to `kustomize build`
	KubeadmConfigPatchesJSON6902 []kustomize.PatchJSON6902
}

// NodeRole defines possible role for nodes in a Kubernetes cluster managed by `kind`
type NodeRole string

const (
	// ControlPlaneRole identifies a node that hosts a Kubernetes control-plane
	// NB. in single node clusters, control-plane nodes act also as a worker nodes
	ControlPlaneRole NodeRole = "control-plane"
	// WorkerRole identifies a node that hosts a Kubernetes worker
	WorkerRole NodeRole = "worker"
	// ExternalEtcdRole identifies a node that hosts an external-etcd instance.
	// Please note that `kind` nodes hosting external etcd are not kubernetes nodes
	ExternalEtcdRole NodeRole = "external-etcd"
	// ExternalLoadBalancerRole identifies a node that hosts an external load balancer for API server
	// in HA configurations.
	// Please note that `kind` nodes hosting external load balancer are not kubernetes nodes
	ExternalLoadBalancerRole NodeRole = "external-load-balancer"
)
