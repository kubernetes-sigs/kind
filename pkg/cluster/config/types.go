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

// NOTE: if you change these types you likely need to update
// Validate() and DeepCopy() at minimum

// Config contains cluster creation config
// This is the current internal config type used by cluster
// Other API versions can be converted to this struct with Convert()
type Config struct {
	// NumNodes is the number of nodes to create (currently only one is supported)
	NumNodes int `json:"numNodes,omitempty"`
	// KubeadmConfigTemplate allows overriding the default template in
	// cluster/kubeadm
	KubeadmConfigTemplate string `json:"kubeadmConfigTemplate,omitempty"`
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
	// PreKubeadm hooks run before `kubeadm`
	PreKubeadm []LifecycleHook `json:"preKubeadm,omitempty"`
	// PostKubeadm hooks run after `kubeadm`
	PostKubeadm []LifecycleHook `json:"postKubeadm,omitempty"`
}

// LifecycleHook represents a command to run at points in the node lifecycle
type LifecycleHook struct {
	// Name is used to improve logging (optional)
	Name string `json:"name,omitempty"`
	// Command is the command to run on the node
	Command string `json:"command"`
	// Args are the arguments to the command (optional)
	Args []string `json:"args,omitempty"`
}
