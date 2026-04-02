/*
Copyright 2024 The Kubernetes Authors.

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

package kubeadm

import (
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		data        ConfigData
		expectError bool
		// substrings that must be present in the output
		mustContain []string
		// substrings that must not be present in the output
		mustNotContain []string
	}{
		{
			name: "basic v1beta3 config (>=v1.23.0)",
			data: ConfigData{
				ClusterName:          "test-cluster",
				KubernetesVersion:    "v1.23.0",
				ControlPlaneEndpoint: "127.0.0.1:6443",
				APIBindPort:          6443,
				APIServerAddress:     "127.0.0.1",
				Token:                Token,
				PodSubnet:            "10.244.0.0/16",
				ServiceSubnet:        "10.96.0.0/12",
				NodeAddress:          "172.17.0.2",
				NodeName:             "test-cluster-control-plane",
				NodeProvider:         "docker",
				KubeProxyMode:        "iptables",
				IPFamily:             config.IPv4Family,
			},
			mustContain: []string{
				"apiVersion: kubeadm.k8s.io/v1beta3",
				"clusterName: \"test-cluster\"",
				"kubernetesVersion: v1.23.0",
				"controlPlaneEndpoint: \"127.0.0.1:6443\"",
				`token: "` + Token + `"`,
				"podSubnet: \"10.244.0.0/16\"",
				"serviceSubnet: \"10.96.0.0/12\"",
			},
			mustNotContain: []string{
				"apiVersion: kubeadm.k8s.io/v1beta2",
			},
		},
		{
			name: "v1beta2 config (<v1.23.0)",
			data: ConfigData{
				ClusterName:          "kind",
				KubernetesVersion:    "v1.22.0",
				ControlPlaneEndpoint: "127.0.0.1:6443",
				APIBindPort:          6443,
				APIServerAddress:     "127.0.0.1",
				Token:                Token,
				PodSubnet:            "10.244.0.0/16",
				ServiceSubnet:        "10.96.0.0/12",
				NodeAddress:          "172.17.0.2",
				NodeName:             "kind-control-plane",
				NodeProvider:         "docker",
				KubeProxyMode:        "iptables",
				IPFamily:             config.IPv4Family,
			},
			mustContain: []string{
				"apiVersion: kubeadm.k8s.io/v1beta2",
			},
			mustNotContain: []string{
				"apiVersion: kubeadm.k8s.io/v1beta3",
			},
		},
		{
			name: "feature gates are included in config",
			data: ConfigData{
				ClusterName:          "kind",
				KubernetesVersion:    "v1.25.0",
				ControlPlaneEndpoint: "127.0.0.1:6443",
				APIBindPort:          6443,
				APIServerAddress:     "127.0.0.1",
				Token:                Token,
				PodSubnet:            "10.244.0.0/16",
				ServiceSubnet:        "10.96.0.0/12",
				NodeAddress:          "172.17.0.2",
				NodeName:             "kind-control-plane",
				NodeProvider:         "docker",
				KubeProxyMode:        "iptables",
				IPFamily:             config.IPv4Family,
				FeatureGates:         map[string]bool{"MyFeature": true},
			},
			mustContain: []string{
				"MyFeature=true",
			},
		},
		{
			name: "invalid kubernetes version returns error",
			data: ConfigData{
				KubernetesVersion: "not-a-version",
			},
			expectError: true,
		},
		{
			name: "rootless provider with incompatible version returns error",
			data: ConfigData{
				KubernetesVersion: "v1.21.0",
				RootlessProvider:  true,
			},
			expectError: true,
		},
		{
			name: "rootless provider with compatible version",
			data: ConfigData{
				ClusterName:          "kind",
				KubernetesVersion:    "v1.22.0",
				ControlPlaneEndpoint: "127.0.0.1:6443",
				APIBindPort:          6443,
				APIServerAddress:     "127.0.0.1",
				Token:                Token,
				PodSubnet:            "10.244.0.0/16",
				ServiceSubnet:        "10.96.0.0/12",
				NodeAddress:          "172.17.0.2",
				NodeName:             "kind-control-plane",
				NodeProvider:         "docker",
				KubeProxyMode:        "iptables",
				IPFamily:             config.IPv4Family,
				RootlessProvider:     true,
			},
			mustContain: []string{
				"KubeletInUserNamespace=true",
			},
		},
		{
			name: "IPv6 family sets IPv6 addresses",
			data: ConfigData{
				ClusterName:          "kind",
				KubernetesVersion:    "v1.25.0",
				ControlPlaneEndpoint: "[::1]:6443",
				APIBindPort:          6443,
				APIServerAddress:     "::1",
				Token:                Token,
				PodSubnet:            "fd00:10:244::/56",
				ServiceSubnet:        "fd00:10:96::/112",
				NodeAddress:          "fc00::2",
				NodeName:             "kind-control-plane",
				NodeProvider:         "docker",
				KubeProxyMode:        "iptables",
				IPFamily:             config.IPv6Family,
			},
			mustContain: []string{
				`address: "::"`,
			},
		},
		{
			name: "kube-proxy config is omitted when mode is none",
			data: ConfigData{
				ClusterName:          "kind",
				KubernetesVersion:    "v1.25.0",
				ControlPlaneEndpoint: "127.0.0.1:6443",
				APIBindPort:          6443,
				APIServerAddress:     "127.0.0.1",
				Token:                Token,
				PodSubnet:            "10.244.0.0/16",
				ServiceSubnet:        "10.96.0.0/12",
				NodeAddress:          "172.17.0.2",
				NodeName:             "kind-control-plane",
				NodeProvider:         "docker",
				KubeProxyMode:        "none",
				IPFamily:             config.IPv4Family,
			},
			mustNotContain: []string{
				"KubeProxyConfiguration",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Config(tc.data)
			assert.ExpectError(t, tc.expectError, err)
			if err != nil {
				return
			}
			for _, s := range tc.mustContain {
				if !strings.Contains(got, s) {
					t.Errorf("Config output does not contain %q\ngot:\n%s", s, got)
				}
			}
			for _, s := range tc.mustNotContain {
				if strings.Contains(got, s) {
					t.Errorf("Config output should not contain %q\ngot:\n%s", s, got)
				}
			}
		})
	}
}
