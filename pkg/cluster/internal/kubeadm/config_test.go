/*
Copyright The Kubernetes Authors.

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
)

func TestConfigVersion(t *testing.T) {
	cases := []struct {
		name              string
		kubernetesVersion string
		expectedVersion   string
	}{
		{
			name:              "v1.22.0 - v1beta2",
			kubernetesVersion: "v1.22.0",
			expectedVersion:   "kubeadm.k8s.io/v1beta2",
		},
		{
			name:              "v1.23.0 - v1beta3",
			kubernetesVersion: "v1.23.0",
			expectedVersion:   "kubeadm.k8s.io/v1beta3",
		},
		{
			name:              "v1.35.0 - v1beta3",
			kubernetesVersion: "v1.35.0",
			expectedVersion:   "kubeadm.k8s.io/v1beta3",
		},
		{
			name:              "v1.36.0 - v1beta4",
			kubernetesVersion: "v1.36.0",
			expectedVersion:   "kubeadm.k8s.io/v1beta4",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := ConfigData{
				KubernetesVersion: tc.kubernetesVersion,
			}
			cfg, err := Config(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(cfg, tc.expectedVersion) {
				t.Errorf("expected config to contain %q, but got:\n%s", tc.expectedVersion, cfg)
			}
		})
	}
}
