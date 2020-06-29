/*
Copyright 2020 The Kubernetes Authors.

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

package kube

import (
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestNewNamedBits(t *testing.T) {
	cases := []struct {
		name          string
		kubeRoot      string
		arch          string
		expected      Bits
		expectedError bool
	}{
		{
			name:     "bazel",
			kubeRoot: "/usr/local",
			arch:     "x86_64",
			expected: &BazelBuildBits{
				kubeRoot: "/usr/local",
				arch:     "x86_64",
				logger:   nil,
			},
		},
		{
			name:     "docker",
			kubeRoot: "/usr/local",
			arch:     "x86_64",
			expected: &DockerBuildBits{
				kubeRoot: "/usr/local",
				arch:     "x86_64",
				logger:   nil,
			},
		},
		{
			name:     "make",
			kubeRoot: "/usr/local",
			arch:     "x86_64",
			expected: &DockerBuildBits{
				kubeRoot: "/usr/local",
				arch:     "x86_64",
				logger:   nil,
			},
		},
		{
			name:          "unknow",
			kubeRoot:      "/usr/local",
			arch:          "x86_64",
			expectedError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NewNamedBits(nil, c.name, c.kubeRoot, c.arch)
			assert.ExpectError(t, c.expectedError, err)
			assert.DeepEqual(t, c.expected, got)
		})
	}
}
