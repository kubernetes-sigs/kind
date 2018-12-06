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

package v1alpha1

import (
	unsafe "unsafe"

	"sigs.k8s.io/kind/pkg/cluster/config"
	kustomize "sigs.k8s.io/kind/pkg/kustomize"
)

// Convert implement a custom conversion func (not using the api machinery)
// that transform v1alpha1 Config into a v1alpha2 Config.
// Using the api machinery for this conversion could be done, but it add several
// constraints to the internal Config object (e.g. add TypeMeta).
// Instead it was preferred to keep the desing of the internal Config clean and simple.
func (in *Config) Convert(out *config.Config) {
	// Internal configuration now supports multinode, so it is necessary to transform
	// v1alpha1 Config into one Node with role control plane and then add it to the list of nodes.
	var node = &config.Node{}
	node.Role = config.ControlPlaneRole
	node.Image = in.Image
	node.KubeadmConfigPatches = *(*[]string)(unsafe.Pointer(&in.KubeadmConfigPatches))
	node.KubeadmConfigPatchesJSON6902 = *(*[]kustomize.PatchJSON6902)(unsafe.Pointer(&in.KubeadmConfigPatchesJSON6902))
	node.ControlPlane = (*config.ControlPlane)(unsafe.Pointer(in.ControlPlane))
	out.Add(node)
}
