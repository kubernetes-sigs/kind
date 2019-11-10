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

package config

import (
	v1alpha3 "sigs.k8s.io/kind/pkg/apis/config/v1alpha3"
)

// Convertv1alpha3 converts a v1alpha3 cluster to a cluster at the internal API version
func Convertv1alpha3(in *v1alpha3.Cluster) *Cluster {
	in = in.DeepCopy() // deep copy first to avoid touching the original
	out := &Cluster{
		Nodes:                        make([]Node, len(in.Nodes)),
		KubeadmConfigPatches:         in.KubeadmConfigPatches,
		KubeadmConfigPatchesJSON6902: make([]PatchJSON6902, len(in.KubeadmConfigPatchesJSON6902)),
	}

	for i := range in.Nodes {
		convertv1alpha3Node(&in.Nodes[i], &out.Nodes[i])
	}

	convertv1alpha3Networking(&in.Networking, &out.Networking)

	for i := range in.KubeadmConfigPatchesJSON6902 {
		convertv1alpha3PatchJSON6902(&in.KubeadmConfigPatchesJSON6902[i], &out.KubeadmConfigPatchesJSON6902[i])
	}

	return out
}

func convertv1alpha3Node(in *v1alpha3.Node, out *Node) {
	out.Role = NodeRole(in.Role)
	out.Image = in.Image

	out.ExtraMounts = make([]Mount, len(in.ExtraMounts))
	out.ExtraPortMappings = make([]PortMapping, len(in.ExtraPortMappings))

	for i := range in.ExtraMounts {
		convertv1alpha3Mount(&in.ExtraMounts[i], &out.ExtraMounts[i])
	}

	for i := range in.ExtraPortMappings {
		convertv1alpha3PortMapping(&in.ExtraPortMappings[i], &out.ExtraPortMappings[i])
	}
}

func convertv1alpha3PatchJSON6902(in *v1alpha3.PatchJSON6902, out *PatchJSON6902) {
	out.Group = in.Group
	out.Version = in.Version
	out.Kind = in.Kind
	// NOTE: name and namespace are discarded, see the docs for the types.
	out.Patch = in.Patch
}

func convertv1alpha3Networking(in *v1alpha3.Networking, out *Networking) {
	out.IPFamily = ClusterIPFamily(in.IPFamily)
	out.APIServerPort = in.APIServerPort
	out.APIServerAddress = in.APIServerAddress
	out.PodSubnet = in.PodSubnet
	out.ServiceSubnet = in.ServiceSubnet
	out.DisableDefaultCNI = in.DisableDefaultCNI
}

func convertv1alpha3Mount(in *v1alpha3.Mount, out *Mount) {
	out.ContainerPath = in.ContainerPath
	out.HostPath = in.HostPath
	out.Readonly = in.Readonly
	out.SelinuxRelabel = in.SelinuxRelabel
	out.Propagation = MountPropagation(v1alpha3.MountPropagationValueToName[in.Propagation])
}

func convertv1alpha3PortMapping(in *v1alpha3.PortMapping, out *PortMapping) {
	out.ContainerPort = in.ContainerPort
	out.HostPort = in.HostPort
	out.ListenAddress = in.ListenAddress
	out.Protocol = PortMappingProtocol(v1alpha3.PortMappingProtocolValueToName[in.Protocol])
}
