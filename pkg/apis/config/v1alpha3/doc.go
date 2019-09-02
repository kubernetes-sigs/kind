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

// Package v1alpha3 implements the v1alpha3 apiVersion of kind's cluster
// configuration
//
// +k8s:deepcopy-gen=package
// +k8s:defaulter-gen=TypeMeta
//
// Package v1alpha3 defines the v1alpha version of the kind configuration file format.
//
// Basics
//
// The preferred way to configure kind is to pass an YAML configuration file with the --config option.
//
// kind supports the following configuration types:
//
//     apiVersion: kind.sigs.k8s.io/v1alpha3
//     kind: Cluster
//
// The Cluster configuration type should be used to configure kind-cluster settings,
// including settings for:
//
// - KubeadmConfigPatches are applied to the generated kubeadm config as
// strategic merge patches to `kustomize build` internally
// https://github.com/kubernetes/community/blob/master/contributors/devel/strategic-merge-patch.md
// This should be an inline yaml blob-string.
//
// - KubeadmConfigPatchesJSON6902 are applied to the generated kubeadm config
// as patchesJson6902 to `kustomize build`.
//
// - Networking contains cluster wide network settings.
//
// - Nodes contains the list of nodes defined in the kind cluster
// If unset this will default to a single control-plane node.
// Note that if more than one control plane is specified, an external
// control plane load balancer will be provisioned implicitly.
//
// Here is a fully populated example of a single YAML file containing multiple
// configuration types to be used during a `kind create cluster` run.
//
//  apiVersion: kind.sigs.k8s.io/v1alpha3
//  kind: Cluster
//  # patch the generated kubeadm config with some extra settings.
//  kubeadmConfigPatches:
//  - |
//    # See  https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2
//    for more details.
//    apiVersion: kubeadm.k8s.io/v1beta2
//    kind: ClusterConfiguration
//    metadata:
//      name: config
//    networking:
//      serviceSubnet: 10.0.0.0/16
//  # patch it further using a JSON 6902 patch.
//  kubeadmConfigPatchesJson6902:
//  - group: kubeadm.k8s.io
//    version: v1beta2
//    kind: ClusterConfiguration
//    patch: |
//    - op: add
//      path: /apiServer/certSANs/-
//      value: my-hostname
//  # IP address on the host to which the Kuberentes API will listen to.
//  networking:
//    apiServerAddress: 1.2.3.4
//  # 1 control plane node and 3 workers.
//  nodes:
//  # the control plane node config.
//  - role: control-plane
//  # the three workers
//  - role: worker
//  - role: worker
//  - role: worker
package v1alpha3
