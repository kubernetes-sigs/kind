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

package cluster

import "testing"

func TestContextValidate(t *testing.T) {
	cases := []struct {
		TestName    string
		Name        string
		ExpectError bool
	}{
		{
			TestName:    "defaults",
			Name:        "",
			ExpectError: false,
		},
		{
			TestName:    "2",
			Name:        "2",
			ExpectError: false,
		},
		{
			TestName:    "foo is fine",
			Name:        "foo",
			ExpectError: false,
		},
		{
			TestName:    "Invalid name - commas",
			Name:        ",,,,,,",
			ExpectError: true,
		},
		{
			TestName:    "Invalid name - invalid character in the middle",
			Name:        "almost-valid@nope",
			ExpectError: true,
		},
		{
			TestName:    "Invalid name - emojis :(",
			Name:        "😬",
			ExpectError: true,
		},
	}

	for _, tc := range cases {
		ctx := NewContext(tc.Name)
		err := ctx.Validate()
		// the error can be:
		// - nil, in which case we should expect no errors or fail
		if err == nil && tc.ExpectError {
			t.Errorf("expected an error for case: '%s'", tc.TestName)
		}
		if err != nil && !tc.ExpectError {
			t.Errorf("did an error for case: '%s' but got error: %v", tc.TestName, err)
		}
	}
}
