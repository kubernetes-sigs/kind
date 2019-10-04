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
	"os"
	"reflect"
	"testing"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

func TestGetProxyEnvs(t *testing.T) {

	tests := []struct {
		name    string
		cluster *config.Cluster
		envs    map[string]string // contains the environment variables to be set for the test
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
			envs: map[string]string{"HTTP_PROXY": "5.5.5.5"},
			want: map[string]string{"HTTP_PROXY": "5.5.5.5", "http_proxy": "5.5.5.5", "NO_PROXY": ",10.0.0.0/24,12.0.0.0/24", "no_proxy": ",10.0.0.0/24,12.0.0.0/24"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			setProxyEnvs(tt.envs)
			if got := GetProxyEnvs(tt.cluster); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetProxyEnvs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func setProxyEnvs(envs map[string]string) {
	for k, v := range envs {
		os.Setenv(k, v)
	}
}
