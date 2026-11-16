/*
Copyright 2026 The Kubernetes Authors.

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

package create

import (
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
	"sigs.k8s.io/kind/pkg/log"
)

func TestValidateProviderInfo(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		info providers.ProviderInfo
		// expectError is a substring the returned error must contain,
		// or "" if no error is expected
		expectError string
	}{
		{
			name: "rootful",
			info: providers.ProviderInfo{
				Rootless: false,
				Cgroup2:  true,
			},
			expectError: "",
		},
		{
			name: "rootful cgroup v1",
			info: providers.ProviderInfo{
				Rootless: false,
				Cgroup2:  false,
			},
			expectError: "",
		},
		{
			name: "rootless with all controllers",
			info: providers.ProviderInfo{
				Rootless:            true,
				Cgroup2:             true,
				SupportsMemoryLimit: true,
				SupportsPidsLimit:   true,
				SupportsCPUShares:   true,
			},
			expectError: "",
		},
		{
			name: "rootless without cgroup v2",
			info: providers.ProviderInfo{
				Rootless: true,
				Cgroup2:  false,
			},
			expectError: "requires cgroup v2",
		},
		{
			name: "rootless with missing cpu controller",
			info: providers.ProviderInfo{
				Rootless:            true,
				Cgroup2:             true,
				SupportsMemoryLimit: true,
				SupportsPidsLimit:   true,
				SupportsCPUShares:   false,
			},
			expectError: "\"Delegate=yes\" to enable the missing cgroup v2 controllers (cpu)",
		},
		{
			name: "rootless with only pids controller",
			info: providers.ProviderInfo{
				Rootless:            true,
				Cgroup2:             true,
				SupportsMemoryLimit: false,
				SupportsPidsLimit:   true,
				SupportsCPUShares:   false,
			},
			expectError: "\"Delegate=yes\" to enable the missing cgroup v2 controllers (cpu, memory)",
		},
		{
			// e.g. inside an LXC container where the root cgroup.controllers
			// is empty, so that setting Delegate=yes cannot help
			// https://github.com/kubernetes-sigs/kind/issues/3868
			name: "rootless without any controllers",
			info: providers.ProviderInfo{
				Rootless:            true,
				Cgroup2:             true,
				SupportsMemoryLimit: false,
				SupportsPidsLimit:   false,
				SupportsCPUShares:   false,
			},
			expectError: "the host may not provide any cgroup v2 controllers",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateProviderInfo(log.NoopLogger{}, &tc.info)
			if tc.expectError == "" {
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q but got none", tc.expectError)
			}
			if !strings.Contains(err.Error(), tc.expectError) {
				t.Fatalf("expected error containing %q but got: %v", tc.expectError, err)
			}
		})
	}
}
