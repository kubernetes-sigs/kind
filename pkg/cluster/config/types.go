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

// +k8s:deepcopy-gen=false

// Config groups all nodes in the `kind` Config.
// This struct is used internally by `kind` and it is NOT EXPOSED as a object of the public API.
// All the field of this type are intentionally defined a private fields, thus ensuring
// that nodes and all the derivedConfigData respect a set of assumptions that will simplify
// the rest of the code e.g. nodes are ordered by provisioning order, node names are
// unique, derivedConfigData are properly set etc.
// Config field can be modified or accessed only using provided helper func.
type Config struct {
	// nodes constains the list of nodes defined in the `kind` Config
	// Such list is not meant to be set by hand, but the Add method
	// should be used instead
	nodes NodeList

	// derivedConfigData is struct populated starting from the node list
	// that provides a set of convenience func for accessing nodes
	// with different role in the kind cluster.
	derivedConfigData
}

// +k8s:deepcopy-gen=false

// NodeList defines a list of Node in the `kind` Config
type NodeList []*Node

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Node contains settings for a node in the `kind` Config.
// A node in kind config represent a container that will be provisioned with all the components
// required for the assigned role in the Kubernetes cluster
type Node struct {
	// TypeMeta representing the type of the object and its API schema version.
	metav1.TypeMeta

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
	// ControlPlane holds config for the control plane node
	ControlPlane *ControlPlane

	// The unique name assigned to the node
	// This information is internal to `kind`.
	// +k8s:conversion-gen=false
	Name string
	// ContainerHandle provides an handle to the container implementing the node
	// This information is internal to `kind`.
	// +k8s:conversion-gen=false
	ContainerHandle
}

// NodeRole defines possible role for nodes in a Kubernetes cluster managed by `kind`
type NodeRole string

const (
	// ControlPlaneRole identifies a node that hosts a Kubernetes control-plane
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

// ControlPlane holds configurations specific to the control plane nodes
// (currently the only node).
type ControlPlane struct {
	// NodeLifecycle contains LifecycleHooks for phases of node provisioning
	NodeLifecycle *NodeLifecycle
}

// NodeLifecycle contains LifecycleHooks for phases of node provisioning
// Within each phase these hooks run in the order specified
type NodeLifecycle struct {
	// PreBoot hooks run before starting systemd
	PreBoot []LifecycleHook
	// PreKubeadm hooks run immediately before `kubeadm`
	PreKubeadm []LifecycleHook
	// PostKubeadm hooks run immediately after `kubeadm`
	PostKubeadm []LifecycleHook
	// PostSetup hooks run after any standard `kind` setup on the node
	PostSetup []LifecycleHook
}

// LifecycleHook represents a command to run at points in the node lifecycle
type LifecycleHook struct {
	// Name is used to improve logging (optional)
	Name string
	// Command is the command to run on the node
	Command []string
	// MustSucceed - if true then the hook / command failing will cause
	// cluster creation to fail, otherwise the error will just be logged and
	// the boot process will continue
	MustSucceed bool
}

// +k8s:deepcopy-gen=false

// derivedConfigData is a struct populated starting from the node list.
// This struct is used internally by `kind` and it is NOT EXPOSED as a object of the public API.
// All the field of this type are intentionally defined a private fields, thus ensuring
// that derivedConfigData respect a set of assumptions that  will simplify the rest of the code.
// derivedConfigData fields can be modified or accessed only using provided helper func.
type derivedConfigData struct {
	// controlPlanes contains the subset of nodes with control-plane role
	controlPlanes NodeList
	// workers contains the subset of nodes with worker role, if any
	workers NodeList
	// externalEtcd contains the node with external-etcd role, if defined
	// TODO(fabriziopandini): eventually in future we would like to support
	// external etcd clusters with more than one member
	externalEtcd *Node
	// externalLoadBalancer contains the node with external-load-balancer role, if defined
	externalLoadBalancer *Node
}

// +k8s:conversion-gen=false

// ContainerHandle defines info used by `kind` for transforming Nodes into containers.
// This struct is used internally by `kind` and it is NOT EXPOSED as a object of the public API.
// TODO(fabriziopandini): this is a place holder for an object that will replace current container handle
// when pkg/cluster/context.go will support multi master
type ContainerHandle struct {
}
