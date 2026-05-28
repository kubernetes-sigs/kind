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
	v1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

// Convertv1alpha4 converts a v1alpha4 cluster to a cluster at the internal API version
func Convertv1alpha4(in *v1alpha4.Cluster) *Cluster {
	in = in.DeepCopy() // deep copy first to avoid touching the original
	out := &Cluster{
		Name:                            in.Name,
		Nodes:                           make([]Node, len(in.Nodes)),
		FeatureGates:                    in.FeatureGates,
		RuntimeConfig:                   in.RuntimeConfig,
		KubeadmConfigPatches:            in.KubeadmConfigPatches,
		KubeadmConfigPatchesJSON6902:    make([]PatchJSON6902, len(in.KubeadmConfigPatchesJSON6902)),
		ContainerdConfigPatches:         in.ContainerdConfigPatches,
		ContainerdConfigPatchesJSON6902: in.ContainerdConfigPatchesJSON6902,
	}

	for i := range in.Nodes {
		convertv1alpha4Node(&in.Nodes[i], &out.Nodes[i])
	}

	// Flatten the optional v1alpha4 `hosts:` block: each host's nested nodes
	// are appended to Cluster.Nodes with their Host field set, and the
	// (context, addr) pair is preserved in Cluster.Hosts so the swarm
	// provider can build its host list from the YAML alone.
	for i := range in.Hosts {
		h := &in.Hosts[i]
		outHost := Host{Context: h.Context, Addr: h.Addr}
		outHost.Nodes = make([]Node, len(h.Nodes))
		for j := range h.Nodes {
			convertv1alpha4Node(&h.Nodes[j], &outHost.Nodes[j])
			outHost.Nodes[j].Host = h.Context
			out.Nodes = append(out.Nodes, outHost.Nodes[j])
		}
		out.Hosts = append(out.Hosts, outHost)
	}

	convertv1alpha4Networking(&in.Networking, &out.Networking)

	for i := range in.KubeadmConfigPatchesJSON6902 {
		convertv1alpha4PatchJSON6902(&in.KubeadmConfigPatchesJSON6902[i], &out.KubeadmConfigPatchesJSON6902[i])
	}

	return out
}

func convertv1alpha4Node(in *v1alpha4.Node, out *Node) {
	out.Role = NodeRole(in.Role)
	out.Image = in.Image
	out.Host = in.Host

	out.Labels = in.Labels
	out.KubeadmConfigPatches = in.KubeadmConfigPatches
	out.ExtraMounts = make([]Mount, len(in.ExtraMounts))
	out.ExtraPortMappings = make([]PortMapping, len(in.ExtraPortMappings))
	out.KubeadmConfigPatchesJSON6902 = make([]PatchJSON6902, len(in.KubeadmConfigPatchesJSON6902))

	for i := range in.ExtraMounts {
		convertv1alpha4Mount(&in.ExtraMounts[i], &out.ExtraMounts[i])
	}

	for i := range in.ExtraPortMappings {
		convertv1alpha4PortMapping(&in.ExtraPortMappings[i], &out.ExtraPortMappings[i])
	}

	for i := range in.KubeadmConfigPatchesJSON6902 {
		convertv1alpha4PatchJSON6902(&in.KubeadmConfigPatchesJSON6902[i], &out.KubeadmConfigPatchesJSON6902[i])
	}
}

func convertv1alpha4PatchJSON6902(in *v1alpha4.PatchJSON6902, out *PatchJSON6902) {
	out.Group = in.Group
	out.Version = in.Version
	out.Kind = in.Kind
	out.Patch = in.Patch
}

func convertv1alpha4Networking(in *v1alpha4.Networking, out *Networking) {
	out.IPFamily = ClusterIPFamily(in.IPFamily)
	out.APIServerPort = in.APIServerPort
	out.APIServerAddress = in.APIServerAddress
	out.PodSubnet = in.PodSubnet
	out.KubeProxyMode = ProxyMode(in.KubeProxyMode)
	out.ServiceSubnet = in.ServiceSubnet
	out.DisableDefaultCNI = in.DisableDefaultCNI
	out.DNSSearch = in.DNSSearch
}

func convertv1alpha4Mount(in *v1alpha4.Mount, out *Mount) {
	out.ContainerPath = in.ContainerPath
	out.HostPath = in.HostPath
	out.Readonly = in.Readonly
	out.SelinuxRelabel = in.SelinuxRelabel
	out.Propagation = MountPropagation(in.Propagation)
}

func convertv1alpha4PortMapping(in *v1alpha4.PortMapping, out *PortMapping) {
	out.ContainerPort = in.ContainerPort
	out.HostPort = in.HostPort
	out.ListenAddress = in.ListenAddress
	out.Protocol = PortMappingProtocol(in.Protocol)
}
