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

package common

import (
	"testing"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestGetProxyEnvs(t *testing.T) {
	t.Parallel()
	// first test the public method
	cfg := &config.Cluster{}
	config.SetDefaultsCluster(cfg)
	envs := GetProxyEnvs(cfg)
	// GetProxyEnvs should always return a valid map
	if envs == nil {
		t.Errorf("GetProxyEnvs returned nil but should not")
	}

	// now test the internal one (with all of the logic)
	cases := []struct {
		name    string
		cluster *config.Cluster
		env     map[string]string
		want    map[string]string
	}{
		{
			name: "No environment variables",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				c.Networking.ServiceSubnet = "10.0.0.0/24"
				c.Networking.PodSubnet = "12.0.0.0/24"
				return &c
			}(),
			want: map[string]string{},
		},
		{
			name: "HTTP_PROXY environment variables",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				c.Networking.ServiceSubnet = "10.0.0.0/24"
				c.Networking.PodSubnet = "12.0.0.0/24"
				return &c
			}(),
			env: map[string]string{
				"HTTP_PROXY": "5.5.5.5",
			},
			want: map[string]string{"HTTP_PROXY": "5.5.5.5", "http_proxy": "5.5.5.5", "NO_PROXY": "10.0.0.0/24,12.0.0.0/24", "no_proxy": "10.0.0.0/24,12.0.0.0/24"},
		},
		{
			name: "HTTPS_PROXY environment variables",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				c.Networking.ServiceSubnet = "10.0.0.0/24"
				c.Networking.PodSubnet = "12.0.0.0/24"
				return &c
			}(),
			env: map[string]string{
				"HTTPS_PROXY": "5.5.5.5",
			},
			want: map[string]string{"HTTPS_PROXY": "5.5.5.5", "https_proxy": "5.5.5.5", "NO_PROXY": "10.0.0.0/24,12.0.0.0/24", "no_proxy": "10.0.0.0/24,12.0.0.0/24"},
		},
		{
			name: "NO_PROXY environment variables",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				c.Networking.ServiceSubnet = "10.0.0.0/24"
				c.Networking.PodSubnet = "12.0.0.0/24"
				return &c
			}(),
			env: map[string]string{
				"HTTPS_PROXY": "5.5.5.5",
				"NO_PROXY":    "8.8.8.8",
			},
			want: map[string]string{"HTTPS_PROXY": "5.5.5.5", "https_proxy": "5.5.5.5", "NO_PROXY": "8.8.8.8,10.0.0.0/24,12.0.0.0/24", "no_proxy": "8.8.8.8,10.0.0.0/24,12.0.0.0/24"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getProxyEnvs(tc.cluster, func(e string) string {
				if tc.env == nil {
					return ""
				}
				return tc.env[e]
			})
			assert.DeepEqual(t, tc.want, result)
		})
	}
}
