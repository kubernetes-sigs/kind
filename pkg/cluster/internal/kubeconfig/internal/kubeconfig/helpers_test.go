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

package kubeconfig

import (
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestKINDClusterKey(t *testing.T) {
	t.Parallel()
	assert.StringEqual(t, "kind-foobar", KINDClusterKey("foobar"))
}

func TestCheckKubeadmExpectations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name        string
		Config      *Config
		ExpectError bool
	}{
		{
			Name: "too many of all entries",
			Config: &Config{
				Clusters: make([]NamedCluster, 5),
				Contexts: make([]NamedContext, 5),
				Users:    make([]NamedUser, 5),
			},
			ExpectError: true,
		},
		{
			Name: "too many users",
			Config: &Config{
				Clusters: make([]NamedCluster, 1),
				Contexts: make([]NamedContext, 1),
				Users:    make([]NamedUser, 2),
			},
			ExpectError: true,
		},
		{
			Name: "too many clusters",
			Config: &Config{
				Clusters: make([]NamedCluster, 2),
				Contexts: make([]NamedContext, 1),
				Users:    make([]NamedUser, 1),
			},
			ExpectError: true,
		},
		{
			Name: "too many contexts",
			Config: &Config{
				Clusters: make([]NamedCluster, 1),
				Contexts: make([]NamedContext, 2),
				Users:    make([]NamedUser, 1),
			},
			ExpectError: true,
		},
		{
			Name: "just right",
			Config: &Config{
				Clusters: make([]NamedCluster, 1),
				Contexts: make([]NamedContext, 1),
				Users:    make([]NamedUser, 1),
			},
			ExpectError: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			assert.ExpectError(t, tc.ExpectError, checkKubeadmExpectations(tc.Config))
		})
	}
}
