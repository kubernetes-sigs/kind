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
	"errors"
	"testing"

	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestGetProxyEnvs(t *testing.T) {
	t.Parallel()

	// now test the internal one (with all of the logic)
	cases := []struct {
		name          string
		cluster       *config.Cluster
		env           map[string]string
		dockerInfoOut string
		dockerInfoErr error
		want          map[string]string
		expectError   bool
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
			name: "No environment variables, but docker config",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				c.Networking.ServiceSubnet = "10.0.0.0/24"
				c.Networking.PodSubnet = "12.0.0.0/24"
				return &c
			}(),
			dockerInfoOut: "HTTP_PROXY=5.5.5.5\nHTTPS_PROXY=5.5.5.5\nNO_PROXY=localhost",
			want:          map[string]string{"HTTPS_PROXY": "5.5.5.5", "https_proxy": "5.5.5.5", "HTTP_PROXY": "5.5.5.5", "http_proxy": "5.5.5.5", "NO_PROXY": "localhost,10.0.0.0/24,12.0.0.0/24", "no_proxy": "localhost,10.0.0.0/24,12.0.0.0/24"},
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
		{
			name: "Invalid docker config",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				c.Networking.ServiceSubnet = "10.0.0.0/24"
				c.Networking.PodSubnet = "12.0.0.0/24"
				return &c
			}(),
			dockerInfoOut: ".....",
			expectError:   true,
		},
		{
			name: "Failed to exec docker info",
			cluster: func() *config.Cluster {
				c := config.Cluster{}
				c.Networking.ServiceSubnet = "10.0.0.0/24"
				c.Networking.PodSubnet = "12.0.0.0/24"
				return &c
			}(),
			dockerInfoErr: errors.New("error"),
			expectError:   true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// fake out getting env
			getEnvFake := func(e string) string {
				if tc.env == nil {
					return ""
				}
				return tc.env[e]
			}
			// fake out docker info --format ...
			getProxyEnvFromDockerFake := &exec.FakeCmder{
				FakeCmd: exec.FakeCmd{
					Out:   []byte(tc.dockerInfoOut),
					Error: tc.dockerInfoErr,
				},
			}
			// actuall test
			result, err := getProxyEnvs(tc.cluster, getEnvFake, getProxyEnvFromDockerFake)
			assert.ExpectError(t, tc.expectError, err)
			assert.DeepEqual(t, tc.want, result)
		})
	}
}
