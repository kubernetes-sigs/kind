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

package docker

import (
	"fmt"
	"testing"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/internal/assert"
)

func Test_generateULASubnetFromName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		attempt int32
		subnet  string
	}{
		{
			name:   "kind",
			subnet: "fc00:f853:ccd:e793::/64",
		},
		{
			name:    "foo",
			attempt: 1,
			subnet:  "fc00:8edf:7f02:ec8f::/64",
		},
		{
			name:    "foo",
			attempt: 2,
			subnet:  "fc00:9968:306b:2c65::/64",
		},
		{
			name:   "kind2",
			subnet: "fc00:444c:147a:44ab::/64",
		},
		{
			name:   "kin",
			subnet: "fc00:fcd9:c2be:8e23::/64",
		},
		{
			name:   "mysupernetwork",
			subnet: "fc00:7ae1:1e0d:b4d4::/64",
		},
	}
	for _, tc := range cases {
		tc := tc // capture variable
		t.Run(fmt.Sprintf("%s,%d", tc.name, tc.attempt), func(t *testing.T) {
			t.Parallel()
			subnet := generateULASubnetFromName(tc.name, tc.attempt)
			if subnet != tc.subnet {
				t.Errorf("Wrong subnet from %v: expected %v, received %v", tc.name, tc.subnet, subnet)
			}
		})
	}
}

func Test_sortNetworkInspectEntries(t *testing.T) {
	cases := []struct {
		Name     string
		Networks []networkInspectEntry
		Sorted   []networkInspectEntry
	}{
		{
			Name: "simple ID sort",
			Networks: []networkInspectEntry{
				{
					ID: "dc7f897c237215c3b73d2c9ba1d4e116d872793a6c1c0e5bf083762998de8b4e",
				},
				{
					ID: "1ed9912325a0d08594ee786de91ebd961e631643877b5ee58ec906b640813eae",
				},
			},
			Sorted: []networkInspectEntry{
				{
					ID: "1ed9912325a0d08594ee786de91ebd961e631643877b5ee58ec906b640813eae",
				},
				{
					ID: "dc7f897c237215c3b73d2c9ba1d4e116d872793a6c1c0e5bf083762998de8b4e",
				},
			},
		},
		{
			Name: "containers attached sort",
			Networks: []networkInspectEntry{
				{
					ID: "1ed9912325a0d08594ee786de91ebd961e631643877b5ee58ec906b640813eae",
				},
				{
					ID: "dc7f897c237215c3b73d2c9ba1d4e116d872793a6c1c0e5bf083762998de8b4e",
					Containers: map[string]map[string]string{
						"a37779e06f3b694eba491dd450aad18bbbaa0a0fce2952e7c9195ea45ae79d41": {
							"Name":       "buildx_buildkit_kind-builder0",
							"EndpointID": "8f6411fb4360059b2f91028f91ef03130abc96d6381afc265ce53c9df89d5a3d",
						},
					},
				},
				{
					ID: "f0445f08b9989921da00250d778975202267fbab364e5fbad0ceb6db24f3f91e",
				},
				{
					ID: "128154205c7d88c7bb9c255d389bc9e222b58a48cf83619976e7665a48e79918",
					Containers: map[string]map[string]string{
						"aad18bbbaa0a0fce2952e7c9195ea45ae79d41a37779e06f3b694eba491dd450": {
							"Name":       "fakey-fake",
							"EndpointID": "f03130abc96d6381afc265ce53c9df89d5a3d8f6411fb4360059b2f91028f91e",
						},
					},
				},
			},
			Sorted: []networkInspectEntry{
				{
					ID: "128154205c7d88c7bb9c255d389bc9e222b58a48cf83619976e7665a48e79918",
					Containers: map[string]map[string]string{
						"aad18bbbaa0a0fce2952e7c9195ea45ae79d41a37779e06f3b694eba491dd450": {
							"Name":       "fakey-fake",
							"EndpointID": "f03130abc96d6381afc265ce53c9df89d5a3d8f6411fb4360059b2f91028f91e",
						},
					},
				},
				{
					ID: "dc7f897c237215c3b73d2c9ba1d4e116d872793a6c1c0e5bf083762998de8b4e",
					Containers: map[string]map[string]string{
						"a37779e06f3b694eba491dd450aad18bbbaa0a0fce2952e7c9195ea45ae79d41": {
							"Name":       "buildx_buildkit_kind-builder0",
							"EndpointID": "8f6411fb4360059b2f91028f91ef03130abc96d6381afc265ce53c9df89d5a3d",
						},
					},
				},
				{
					ID: "1ed9912325a0d08594ee786de91ebd961e631643877b5ee58ec906b640813eae",
				},
				{
					ID: "f0445f08b9989921da00250d778975202267fbab364e5fbad0ceb6db24f3f91e",
				},
			},
		},
		{
			Name: "network with containers listed after one without",
			Networks: []networkInspectEntry{
				{
					ID: "dc7f897c237215c3b73d2c9ba1d4e116d872793a6c1c0e5bf083762998de8b4e",
					Containers: map[string]map[string]string{
						"a37779e06f3b694eba491dd450aad18bbbaa0a0fce2952e7c9195ea45ae79d41": {
							"Name": "kind-control-plane",
						},
					},
				},
				{
					ID: "1ed9912325a0d08594ee786de91ebd961e631643877b5ee58ec906b640813eae",
				},
			},
			Sorted: []networkInspectEntry{
				{
					ID: "dc7f897c237215c3b73d2c9ba1d4e116d872793a6c1c0e5bf083762998de8b4e",
					Containers: map[string]map[string]string{
						"a37779e06f3b694eba491dd450aad18bbbaa0a0fce2952e7c9195ea45ae79d41": {
							"Name": "kind-control-plane",
						},
					},
				},
				{
					ID: "1ed9912325a0d08594ee786de91ebd961e631643877b5ee58ec906b640813eae",
				},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// the result must not depend on the order docker lists networks in
			for _, networks := range permutations(tc.Networks) {
				sortNetworkInspectEntries(networks)
				assert.DeepEqual(t, tc.Sorted, networks)
			}
		})
	}
}

// permutations returns every ordering of entries, each as a new slice
func permutations(entries []networkInspectEntry) [][]networkInspectEntry {
	if len(entries) <= 1 {
		return [][]networkInspectEntry{append([]networkInspectEntry{}, entries...)}
	}
	var result [][]networkInspectEntry
	for i := range entries {
		rest := make([]networkInspectEntry, 0, len(entries)-1)
		rest = append(rest, entries[:i]...)
		rest = append(rest, entries[i+1:]...)
		for _, p := range permutations(rest) {
			result = append(result, append([]networkInspectEntry{entries[i]}, p...))
		}
	}
	return result
}

func Test_isPoolOverlapError(t *testing.T) {
	cases := []struct {
		Name     string
		Err      error
		Expected bool
	}{
		{
			Name:     "nil error",
			Err:      nil,
			Expected: false,
		},
		{
			Name:     "not a RunError",
			Err:      errors.New("failed to decode networks list"),
			Expected: false,
		},
		{
			Name: "pool overlap",
			Err: errors.WithStack(&exec.RunError{
				Inner:  errors.New("exit status 1"),
				Output: []byte("Error response from daemon: Pool overlaps with other one on this address space"),
			}),
			Expected: true,
		},
		{
			Name: "networks have overlapping",
			Err: &exec.RunError{
				Inner:  errors.New("exit status 1"),
				Output: []byte("Error response from daemon: invalid pool request: Pool overlaps: networks have overlapping IPv6"),
			},
			Expected: true,
		},
		{
			Name: "unrelated RunError",
			Err: &exec.RunError{
				Inner:  errors.New("exit status 1"),
				Output: []byte("Error response from daemon: network with name kind already exists"),
			},
			Expected: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			assert.BoolEqual(t, tc.Expected, isPoolOverlapError(tc.Err))
		})
	}
}
