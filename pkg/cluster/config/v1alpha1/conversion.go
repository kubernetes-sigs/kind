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
	"fmt"
	unsafe "unsafe"

	conversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/kind/pkg/cluster/config"
	kustomize "sigs.k8s.io/kind/pkg/kustomize"
)

func Convert_v1alpha1_Config_To_config_Config(in *Config, out *config.Config, s conversion.Scope) error {
	if err := autoConvert_v1alpha1_Config_To_config_Config(in, out, s); err != nil {
		return err
	}

	// converts v1alpha1 Config into an internal config with a single control-plane node
	var node config.Node

	node.Role = config.ControlPlaneRole
	node.Image = in.Image
	node.KubeadmConfigPatches = *(*[]string)(unsafe.Pointer(&in.KubeadmConfigPatches))
	node.KubeadmConfigPatchesJSON6902 = *(*[]kustomize.PatchJSON6902)(unsafe.Pointer(&in.KubeadmConfigPatchesJSON6902))
	node.ControlPlane = (*config.ControlPlane)(unsafe.Pointer(in.ControlPlane))

	out.Nodes = []config.Node{node}

	return nil
}

func Convert_config_Config_To_v1alpha1_Config(in *config.Config, out *Config, s conversion.Scope) error {
	if err := autoConvert_config_Config_To_v1alpha1_Config(in, out, s); err != nil {
		return err
	}

	// convertion from internal config to v1alpha1 Config is used only by the fuzzer roundtrip test;
	// the fuzzer is configured in order to enforce the number and type of nodes to get always the
	// following condition pass

	if len(in.Nodes) > 1 {
		return fmt.Errorf("invalid conversion. `kind` config with more than one Node cannot be converted to v1alpha1 config format")
	}

	var node = in.Nodes[0]

	if !node.IsControlPlane() {
		return fmt.Errorf("invalid conversion. `kind` config without a control-plane Node cannot be converted to v1alpha1 config format %v", node)
	}

	out.Image = node.Image
	out.KubeadmConfigPatches = *(*[]string)(unsafe.Pointer(&node.KubeadmConfigPatches))
	out.KubeadmConfigPatchesJSON6902 = *(*[]kustomize.PatchJSON6902)(unsafe.Pointer(&node.KubeadmConfigPatchesJSON6902))
	out.ControlPlane = (*ControlPlane)(unsafe.Pointer(node.ControlPlane))

	return nil
}
