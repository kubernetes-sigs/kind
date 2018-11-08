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
	"sigs.k8s.io/kind/pkg/kustomize"
)

// NOTE: if you change these types you likely need to update
// Validate() and DeepCopy() at minimum

// Config contains cluster creation config
// This is the current internal config type used by cluster
// Other API versions can be converted to this struct with Convert()
type Config struct {
	// Image is the node image to use when running the cluster
	// TODO(bentheelder): split this into image and tag?
	Image string `json:"image,omitempty"`
	// NumNodes is the number of nodes to create (currently only one is supported)
	NumNodes int `json:"numNodes,omitempty"`
	// KubeadmConfigPatches are applied to the generated kubeadm config as
	// strategic merge patches to `kustomize build` internally
	// https://github.com/kubernetes/community/blob/master/contributors/devel/strategic-merge-patch.md
	// This should be an inline yaml blob-string
	KubeadmConfigPatches []string `json:"kubeadmConfigPatches,omitempty"`
	// KubeadmConfigPatchesJSON6902 are applied to the generated kubeadm config
	// as patchesJson6902 to `kustomize build`
	KubeadmConfigPatchesJSON6902 []kustomize.PatchJSON6902 `json:"kubeadmConfigPatchesJson6902,omitempty"`
	// ControlPlane holds config for the control plane node
	ControlPlane *ControlPlane `json:"ControlPlane,omitempty"`
}

type ControlPlane struct {
	// NodeLifecycle contains LifecycleHooks for phases of node provisioning
	NodeLifecycle *NodeLifecycle `json:"nodeLifecycle,omitempty"`
}

// ensure current version implements the common interface for
// conversion, validation, etc.
var _ Any = &Config{}

// NodeLifecycle contains LifecycleHooks for phases of node provisioning
// Within each phase these hooks run in the order specified
type NodeLifecycle struct {
	// PreBoot hooks run before starting systemd
	PreBoot []LifecycleHook `json:"preBoot,omitempty"`
	// PreKubeadm hooks run immediately before `kubeadm`
	PreKubeadm []LifecycleHook `json:"preKubeadm,omitempty"`
	// PostKubeadm hooks run immediately after `kubeadm`
	PostKubeadm []LifecycleHook `json:"postKubeadm,omitempty"`
	// PostSetup hooks run after any standard `kind` setup on the node
	PostSetup []LifecycleHook `json:"postSetup,omitempty"`
}

// LifecycleHook represents a command to run at points in the node lifecycle
type LifecycleHook struct {
	// Name is used to improve logging (optional)
	Name string `json:"name,omitempty"`
	// Command is the command to run on the node
	Command []string `json:"command"`
	// MustSucceed - if true then the hook / command failing will cause
	// cluster creation to fail, otherwise the error will just be logged and
	// the boot process will continue
	MustSucceed bool `json:"mustSucceed,omitempty"`
}
