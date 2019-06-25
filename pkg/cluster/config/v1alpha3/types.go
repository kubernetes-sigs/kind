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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kind/pkg/container/cri"
	"sigs.k8s.io/kind/pkg/kustomize"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster contains kind cluster configuration
type Cluster struct {
	// TypeMeta representing the type of the object and its API schema version.
	metav1.TypeMeta `json:",inline"`

	// Nodes contains the list of nodes defined in the `kind` Cluster
	// If unset this will default to a single control-plane node
	// Note that if more than one control plane is specified, an external
	// control plane load balancer will be provisioned implicitly
	Nodes []Node `json:"nodes"`

	/* Advanced fields */

	// Networking contains cluster wide network settings
	Networking Networking `json:"networking"`

	// KubeadmConfigPatches are applied to the generated kubeadm config as
	// strategic merge patches to `kustomize build` internally
	// https://github.com/kubernetes/community/blob/master/contributors/devel/strategic-merge-patch.md
	// This should be an inline yaml blob-string
	KubeadmConfigPatches []string `json:"kubeadmConfigPatches,omitempty"`

	// KubeadmConfigPatchesJSON6902 are applied to the generated kubeadm config
	// as patchesJson6902 to `kustomize build`
	KubeadmConfigPatchesJSON6902 []kustomize.PatchJSON6902 `json:"kubeadmConfigPatchesJson6902,omitempty"`
}

// Node contains settings for a node in the `kind` Cluster.
// A node in kind config represent a container that will be provisioned with all the components
// required for the assigned role in the Kubernetes cluster
type Node struct {
	// Role defines the role of the node in the in the Kubernetes cluster
	// created by kind
	//
	// Defaults to "control-plane"
	Role NodeRole `json:"role,omitempty"`

	// Image is the node image to use when creating this node
	// If unset a default image will be used, see defaults.Image
	Image string `json:"image,omitempty"`

	/* Advanced fields */

	// ExtraMounts describes additional mount points for the node container
	// These may be used to bind a hostPath
	ExtraMounts []cri.Mount `json:"extraMounts,omitempty"`

	// ExtraPortMappings describes additional port mappings for the node container
	// binded to a host Port
	ExtraPortMappings []cri.PortMapping `json:"extraPortMappings,omitempty"`
}

// NodeRole defines possible role for nodes in a Kubernetes cluster managed by `kind`
type NodeRole string

const (
	// ControlPlaneRole identifies a node that hosts a Kubernetes control-plane.
	// NOTE: in single node clusters, control-plane nodes act also as a worker
	// nodes, in which case the taint will be removed. see:
	// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#control-plane-node-isolation
	ControlPlaneRole NodeRole = "control-plane"
	// WorkerRole identifies a node that hosts a Kubernetes worker
	WorkerRole NodeRole = "worker"
)

// Networking contains cluster wide network settings
type Networking struct {
	// IPFamily is the network cluster model, currently it can be ipv4 or ipv6
	IPFamily ClusterIPFamily `json:"ipFamily,omitempty"`
	// APIServerPort is the listen port on the host for the Kubernetes API Server
	// Defaults to a random port on the host
	APIServerPort int32 `json:"apiServerPort,omitempty"`
	// APIServerAddress is the listen address on the host for the Kubernetes
	// API Server. This should be an IP address.
	//
	// Defaults to 127.0.0.1
	APIServerAddress string `json:"apiServerAddress,omitempty"`
	// PodSubnet is the CIDR used for pod IPs
	// kind will select a default if unspecified
	PodSubnet string `json:"podSubnet,omitempty"`
	// ServiceSubnet is the CIDR used for services VIPs
	// kind will select a default if unspecified for IPv6
	ServiceSubnet string `json:"serviceSubnet,omitempty"`
	// If DisableDefaultCNI is true, kind will not install the default CNI setup.
	// Instead the user should install their own CNI after creating the cluster.
	DisableDefaultCNI bool `json:"disableDefaultCNI,omitempty"`
}

// ClusterIPFamily defines cluster network IP family
type ClusterIPFamily string

const (
	// IPv4Family sets ClusterIPFamily to ipv4
	IPv4Family ClusterIPFamily = "ipv4"
	// IPv6Family sets ClusterIPFamily to ipv6
	IPv6Family ClusterIPFamily = "ipv6"
)
