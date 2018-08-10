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

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		Name           string
		Config         Config
		ExpectedErrors int
	}{
		{
			Name:           "Canonical config",
			Config:         NewConfig("1"),
			ExpectedErrors: 0,
		},
		{
			Name: "Invalid name - commas",
			Config: Config{
				Name: ",,,,,,",
			},
			ExpectedErrors: 1,
		},
		{
			Name: "Invalid name - zero length",
			Config: Config{
				Name: "",
			},
			ExpectedErrors: 1,
		},
		{
			Name: "Invalid name - invalid character in the middle",
			Config: Config{
				Name: "almost-valid@nope",
			},
			ExpectedErrors: 1,
		},
	}

	for _, tc := range cases {
		err := tc.Config.Validate()
		// the error can be:
		// - nil, in which case we should expect no errors or fail
		if err == nil {
			if tc.ExpectedErrors != 0 {
				t.Errorf("received no errors but expected errors for case %s", tc.Name)
			}
			continue
		}
		// - not castable to ConfigErrors, in which case we have the wrong error type ...
		configErrors, ok := err.(ConfigErrors)
		if !ok {
			t.Errorf("config.Validate should only return nil or ConfigErrors{...}, got: %v for case: %s", err, tc.Name)
			continue
		}
		// - ConfigErrors, in which case expect a certain number of errors
		errors := configErrors.Errors()
		if len(errors) != tc.ExpectedErrors {
			t.Errorf("expected %d errors but got len(%v) = %d for case: %s", tc.ExpectedErrors, errors, len(errors), tc.Name)
		}
	}
}
