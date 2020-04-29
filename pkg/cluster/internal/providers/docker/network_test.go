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
	"testing"
)

func Test_generateULASubnetFromName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		subnet string
	}{
		{
			name:   "kind",
			subnet: "fc00:5ae0:f40:c847::/64",
		},
		{
			name:   "kind2",
			subnet: "fc00:cadf:7baf:98ea::/64",
		},
		{
			name:   "kin",
			subnet: "fc00:f804:3187:7254::/64",
		},
		{
			name:   "mysupernetwork",
			subnet: "fc00:1dd9:697c:314e::/64",
		},
	}
	for _, tc := range cases {
		tc := tc // capture variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			subnet := generateULASubnetFromName(tc.name)
			if subnet != tc.subnet {
				t.Errorf("Wrong subnet from %v: expected %v, received %v", tc.name, tc.subnet, subnet)
			}

		})
	}
}
