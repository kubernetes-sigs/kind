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
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/kind/pkg/cluster/config/defaults"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_Cluster sets uninitialized fields to their default value.
func SetDefaults_Cluster(obj *Cluster) {
	// default to a one node cluster
	if len(obj.Nodes) == 0 {
		obj.Nodes = []Node{
			{
				Image: defaults.Image,
				Role:  ControlPlaneRole,
			},
		}
	}
	// default to listening on 127.0.0.1:randomPort
	// TODO(bentheelder): this defaulting will need to be ipv6 aware as well
	if obj.Networking.APIServerAddress == "" {
		obj.Networking.APIServerAddress = "127.0.0.1"
	}
}

// SetDefaults_Node sets uninitialized fields to their default value.
func SetDefaults_Node(obj *Node) {
	if obj.Image == "" {
		obj.Image = defaults.Image
	}

	if obj.Role == "" {
		obj.Role = ControlPlaneRole
	}
}
