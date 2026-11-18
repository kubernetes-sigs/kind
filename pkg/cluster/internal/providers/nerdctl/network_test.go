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

package nerdctl

import (
	"fmt"
	"testing"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
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

func Test_isPoolOverlapError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "not a RunError",
			err:      errors.New("something else failed"),
			expected: false,
		},
		{
			name: "pool overlap",
			err: errors.WithStack(&exec.RunError{
				Inner:  errors.New("exit status 1"),
				Output: []byte("Error response from daemon: Pool overlaps with other one on this address space"),
			}),
			expected: true,
		},
		{
			name: "networks have overlapping",
			err: &exec.RunError{
				Inner:  errors.New("exit status 1"),
				Output: []byte("networks have overlapping IPv6"),
			},
			expected: true,
		},
		{
			name: "unrelated RunError",
			err: &exec.RunError{
				Inner:  errors.New("exit status 1"),
				Output: []byte("network with name kind already exists"),
			},
			expected: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isPoolOverlapError(tc.err); got != tc.expected {
				t.Errorf("isPoolOverlapError() = %v, expected %v", got, tc.expected)
			}
		})
	}
}
