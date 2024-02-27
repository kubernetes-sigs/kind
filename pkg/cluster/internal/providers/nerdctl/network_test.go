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
