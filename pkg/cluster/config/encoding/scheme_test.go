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
	"reflect"
	"testing"
)

// TODO(fabrizio pandini): once we have multiple config API versions we
// will need more tests

func TestLoadCurrent(t *testing.T) {
	cases := []struct {
		Name        string
		Path        string
		ExpectError bool
	}{
		{
			Name:        "valid minimal",
			Path:        "./testdata/valid-minimal.yaml",
			ExpectError: false,
		},
		{
			Name:        "valid with lifecyclehooks",
			Path:        "./testdata/valid-with-lifecyclehooks.yaml",
			ExpectError: false,
		},
		{
			Name:        "invalid path",
			Path:        "./testdata/not-a-file.bogus",
			ExpectError: true,
		},
		{
			Name:        "invalid apiVersion",
			Path:        "./testdata/invalid-apiversion.yaml",
			ExpectError: true,
		},
		{
			Name:        "invalid yaml",
			Path:        "./testdata/invalid-yaml.yaml",
			ExpectError: true,
		},
	}
	for _, tc := range cases {
		_, err := Load(tc.Path)
		if err != nil && !tc.ExpectError {
			t.Errorf("case: '%s' got error loading and expected none: %v", tc.Name, err)
		} else if err == nil && tc.ExpectError {
			t.Errorf("case: '%s' got no error loading but expected one", tc.Name)
		}
	}
}

func TestLoadDefault(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Errorf("got error loading default config but expected none: %v", err)
		t.FailNow()
	}
	defaultConfig := newDefaultedConfig()
	if !reflect.DeepEqual(cfg, defaultConfig) {
		t.Errorf(
			"Load(\"\") should match config.New() but does not: %v != %v",
			cfg, defaultConfig,
		)
	}
}
