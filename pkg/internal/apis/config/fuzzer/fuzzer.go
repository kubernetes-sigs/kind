/*
Copyright 2017 The Kubernetes Authors.

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

package fuzzer

import (
	fuzz "github.com/google/gofuzz"

	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// Funcs returns custom fuzzer functions for the `kind` Config.
func Funcs(codecs runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		fuzzConfig,
		fuzzNode,
	}
}

func fuzzConfig(obj *config.Cluster, c fuzz.Continue) {
	c.FuzzNoCustom(obj)

	// Pinning values for fields that get defaults if fuzz value is empty string or nil
	obj.Nodes = []config.Node{{
		Image: "foo:bar",
		Role:  config.ControlPlaneRole,
	}}
	obj.Networking.APIServerAddress = "127.0.0.1"
	obj.Networking.PodSubnet = "10.244.0.0/16"
	obj.Networking.ServiceSubnet = "10.96.0.0/12"
	obj.Networking.IPFamily = "ipv4"
}

func fuzzNode(obj *config.Node, c fuzz.Continue) {
	c.FuzzNoCustom(obj)

	// Pinning values for fields that get defaults if fuzz value is empty string or nil
	obj.Image = "foo:bar"
	obj.Role = config.ControlPlaneRole
}
