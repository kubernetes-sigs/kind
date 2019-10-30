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
	"reflect"
	"testing"

	"sigs.k8s.io/kind/pkg/internal/util/assert"
)

func TestMerge(t *testing.T) {
	cases := []struct {
		Name        string
		Existing    *Config
		Kind        *Config
		Expected    *Config
		ExpectError bool
	}{
		{
			Name:        "bad kind config",
			Existing:    &Config{},
			Kind:        &Config{},
			Expected:    &Config{},
			ExpectError: true,
		},
		{
			Name:     "empty existing",
			Existing: &Config{},
			Kind: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Expected: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			ExpectError: false,
		},
		{
			Name: "replace existing",
			Existing: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
						Cluster: Cluster{
							Server: "foo",
						},
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Kind: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Expected: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			ExpectError: false,
		},
		{
			Name: "add to existing",
			Existing: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kops-blah",
						Cluster: Cluster{
							Server: "foo",
						},
					},
				},
				Users: []NamedUser{
					{
						Name: "kops-blah",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kops-blah",
					},
				},
			},
			Kind: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Expected: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kops-blah",
						Cluster: Cluster{
							Server: "foo",
						},
					},
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kops-blah",
					},
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kops-blah",
					},
					{
						Name: "kind-kind",
					},
				},
			},
			ExpectError: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			err := merge(tc.Existing, tc.Kind)
			assert.ExpectError(t, tc.ExpectError, err)
			if !tc.ExpectError && !reflect.DeepEqual(tc.Existing, tc.Expected) {
				t.Errorf("Merged Config did not equal Expected")
				t.Errorf("Expected: %+v", tc.Expected)
				t.Errorf("Actual: %+v", tc.Existing)
			}
		})
	}
}
