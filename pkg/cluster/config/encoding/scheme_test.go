/*
Copyright 2018 The Kubernetes Authors.

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

package encoding

import (
	"testing"
)

func TestLoadCurrent(t *testing.T) {
	cases := []struct {
		TestName    string
		Path        string
		ExpectError bool
	}{
		{
			TestName:    "no config",
			Path:        "",
			ExpectError: false,
		},
		{
			TestName:    "v1alpha3 minimal",
			Path:        "./testdata/v1alpha3/valid-minimal.yaml",
			ExpectError: false,
		},
		{
			TestName:    "v1alpha3 config with 2 nodes",
			Path:        "./testdata/v1alpha3/valid-minimal-two-nodes.yaml",
			ExpectError: false,
		},
		{
			TestName:    "v1alpha3 full HA",
			Path:        "./testdata/v1alpha3/valid-full-ha.yaml",
			ExpectError: false,
		},
		{
			TestName:    "invalid path",
			Path:        "./testdata/not-a-file.bogus",
			ExpectError: true,
		},
		{
			TestName:    "Invalid apiversion",
			Path:        "./testdata/invalid-apiversion.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid kind",
			Path:        "./testdata/invalid-kind.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid yaml",
			Path:        "./testdata/invalid-yaml.yaml",
			ExpectError: true,
		},
	}
	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			_, err := Load(c.Path)

			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t.Fatalf("unexpected error while Loading config: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if c.ExpectError {
				t.Fatalf("unexpected lack or error while Loading config")
			}
		})
	}
}
