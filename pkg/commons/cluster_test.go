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

package commons

import (
	"testing"
)

func TestGetPrefixedRegistryURL(t *testing.T) {
	baseURL := "123456789.dkr.ecr.eu-west-1.amazonaws.com"

	tests := []struct {
		name                 string
		originalRegistry     string
		baseRegistryURL      string
		ecrPullThroughCacheEnabled bool
		expected             string
	}{
		// CentralECR enabled — all 5 known registry mappings
		{
			name:                 "docker.io maps to /dockerhub suffix",
			originalRegistry:     "docker.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: true,
			expected:             baseURL + DefaultDockerhubPrefix,
		},
		{
			name:                 "public.ecr.aws maps to /ecrpublic suffix",
			originalRegistry:     "public.ecr.aws",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: true,
			expected:             baseURL + DefaultEcrpublicPrefix,
		},
		{
			name:                 "ghcr.io maps to /ghcr suffix",
			originalRegistry:     "ghcr.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: true,
			expected:             baseURL + DefaultGhcrPrefix,
		},
		{
			name:                 "quay.io maps to /quay suffix",
			originalRegistry:     "quay.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: true,
			expected:             baseURL + DefaultQuayPrefix,
		},
		{
			name:                 "k8s.io maps to /k8s suffix",
			originalRegistry:     "k8s.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: true,
			expected:             baseURL + DefaultK8sPrefix,
		},
		// registry.k8s.io also matches the k8s.io case via strings.Contains
		{
			name:                 "registry.k8s.io maps to /k8s suffix via strings.Contains",
			originalRegistry:     "registry.k8s.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: true,
			expected:             baseURL + DefaultK8sPrefix,
		},
		// Unknown registry returns base URL unchanged
		{
			name:                 "unknown registry returns base URL unchanged",
			originalRegistry:     "mcr.microsoft.com",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: true,
			expected:             baseURL,
		},
		// CentralECR disabled — always returns base URL unchanged
		{
			name:                 "CentralECR disabled returns base URL for docker.io",
			originalRegistry:     "docker.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: false,
			expected:             baseURL,
		},
		{
			name:                 "CentralECR disabled returns base URL for quay.io",
			originalRegistry:     "quay.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: false,
			expected:             baseURL,
		},
		{
			name:                 "CentralECR disabled returns base URL for registry.k8s.io",
			originalRegistry:     "registry.k8s.io",
			baseRegistryURL:      baseURL,
			ecrPullThroughCacheEnabled: false,
			expected:             baseURL,
		},
		// Empty base URL returns empty string
		{
			name:                 "empty base URL returns empty string when CentralECR enabled",
			originalRegistry:     "docker.io",
			baseRegistryURL:      "",
			ecrPullThroughCacheEnabled: true,
			expected:             "",
		},
		{
			name:                 "empty base URL returns empty string when CentralECR disabled",
			originalRegistry:     "docker.io",
			baseRegistryURL:      "",
			ecrPullThroughCacheEnabled: false,
			expected:             "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPrefixedRegistryURL(tt.originalRegistry, tt.baseRegistryURL, tt.ecrPullThroughCacheEnabled)
			if got != tt.expected {
				t.Errorf("GetPrefixedRegistryURL(%q, %q, %v) = %q, want %q",
					tt.originalRegistry, tt.baseRegistryURL, tt.ecrPullThroughCacheEnabled, got, tt.expected)
			}
		})
	}
}
