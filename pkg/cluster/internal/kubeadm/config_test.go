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

func TestConfigControlPlaneKubeletLocalMode(t *testing.T) {
	cases := []struct {
		name              string
		kubernetesVersion string
		expectGate        bool
	}{
		{
			// The gate doesn't exist yet.
			name:              "v1.30.0 - no gate",
			kubernetesVersion: "v1.30.0",
			expectGate:        false,
		},
		{
			// Alpha, disabled by default: kind must enable it so
			// control plane kubelets use the local apiserver.
			name:              "v1.31.0 - gate enabled",
			kubernetesVersion: "v1.31.0",
			expectGate:        true,
		},
		{
			name:              "v1.32.0 - gate enabled",
			kubernetesVersion: "v1.32.0",
			expectGate:        true,
		},
		{
			// Beta and enabled by default since v1.33.
			name:              "v1.33.0 - no gate",
			kubernetesVersion: "v1.33.0",
			expectGate:        false,
		},
		{
			// The gate was removed in v1.36; setting it would be an error.
			name:              "v1.36.0 - no gate",
			kubernetesVersion: "v1.36.0",
			expectGate:        false,
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
			hasGate := strings.Contains(cfg, "\"ControlPlaneKubeletLocalMode\": true")
			if hasGate != tc.expectGate {
				t.Errorf("expected ControlPlaneKubeletLocalMode gate presence to be %v, but got:\n%s", tc.expectGate, cfg)
			}
		})
	}
}
