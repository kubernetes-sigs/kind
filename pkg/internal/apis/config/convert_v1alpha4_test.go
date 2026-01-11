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
	"testing"

	v1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

func TestConvertv1alpha4_DisableDefaultStorageClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		input    *v1alpha4.Cluster
		expected bool
	}{
		{
			name: "DisableDefaultStorageClass is false by default",
			input: &v1alpha4.Cluster{
				Nodes: []v1alpha4.Node{
					{Role: v1alpha4.ControlPlaneRole},
				},
			},
			expected: false,
		},
		{
			name: "DisableDefaultStorageClass is true when set",
			input: &v1alpha4.Cluster{
				Nodes: []v1alpha4.Node{
					{Role: v1alpha4.ControlPlaneRole},
				},
				DisableDefaultStorageClass: true,
			},
			expected: true,
		},
		{
			name: "DisableDefaultStorageClass is false when explicitly set",
			input: &v1alpha4.Cluster{
				Nodes: []v1alpha4.Node{
					{Role: v1alpha4.ControlPlaneRole},
				},
				DisableDefaultStorageClass: false,
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := Convertv1alpha4(tc.input)
			if result.DisableDefaultStorageClass != tc.expected {
				t.Errorf("expected DisableDefaultStorageClass to be %v, got %v", tc.expected, result.DisableDefaultStorageClass)
			}
		})
	}
}
