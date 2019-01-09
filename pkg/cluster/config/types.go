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

	// DerivedConfigData is struct populated starting from the node list.
	// It contains a set of "materialized views" generated starting from nodes
	// and designed to make easy operating nodes in the rest of the code base.
	// This attribute exists only in the internal config version and is meant
	// to simplify the usage of the config in the code base.
	// TODO(fabrizio pandini): consider if we can move this away from the api
	// and make it an internal of the kind library
	// +k8s:conversion-gen=false
	DerivedConfigData
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
	// ControlPlane holds config for the control plane node
	ControlPlane *ControlPlane
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

// +k8s:conversion-gen=false

// NodeReplica defines a `kind` config Node that is geneated by creating a replicas for a node
// This attribute exists only in the internal config version and is meant
// to simplify the usage of the config in the code base.
type NodeReplica struct {
	// Node contains settings for the node in the `kind` Config.
	// please note that the Replicas number is alway set to nil.
	Node

	// Name contains the unique name assigned to the node while generating the replica
	Name string
}

// +k8s:conversion-gen=false

// ReplicaList defines a list of NodeReplicas in the `kind` Config
// This attribute exists only in the internal config version and is meant
// to simplify the usage of the config in the code base.
type ReplicaList []*NodeReplica

// +k8s:conversion-gen=false

// DerivedConfigData is a struct populated starting from the nodes list.
// All the field of this type are intentionally defined a private fields, thus ensuring
// that derivedConfigData will have a predictable behaviour for the code built on top.
// This attribute exists only in the internal config version and is meant
// to simplify the usage of the config in the code base.
type DerivedConfigData struct {
	// allReplicas constains the list of node replicas defined in the `kind` Config
	allReplicas ReplicaList
	// controlPlanes contains the subset of node replicas with control-plane role
	controlPlanes ReplicaList
	// workers contains the subset of node replicas with worker role, if any
	workers ReplicaList
	// externalEtcd contains the node replica with external-etcd role, if defined
	// TODO(fabriziopandini): eventually in future we would like to support
	// external etcd clusters with more than one member
	externalEtcd *NodeReplica
	// externalLoadBalancer contains the node replica with external-load-balancer role, if defined
	externalLoadBalancer *NodeReplica
}
