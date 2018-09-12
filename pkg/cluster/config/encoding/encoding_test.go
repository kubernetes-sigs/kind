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

	"sigs.k8s.io/kind/pkg/cluster/config"
)

// TODO(bentheelder): once we have multiple config API versions we
// will need more tests for Load and LoadCurrent

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
		_, err := LoadCurrent(tc.Path)
		if err != nil && !tc.ExpectError {
			t.Errorf("case: '%s' got error`Load`ing and expected none: %v", tc.Name, err)
		} else if err == nil && tc.ExpectError {
			t.Errorf("case: '%s' got no error `Load`ing but expected one", tc.Name)
		}
	}
}

func TestLoadDefault(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Errorf("got error `Load`ing default config but expected none: %v", err)
		t.FailNow()
	}
	defaultConfig := config.New()
	if !reflect.DeepEqual(cfg, defaultConfig) {
		t.Errorf(
			"Load(\"\") should match config.New() but does not: %v != %v",
			cfg, defaultConfig,
		)
		t.FailNow()
	}
}

func TestEncodingRoundTrip(t *testing.T) {
	cfg := config.New()
	marshaled, err := Marshal(cfg)
	if err != nil {
		t.Errorf("got error `Marshal`ing default config: %v", err)
		t.FailNow()
	}
	roundTripConfig, err := Unmarshal(marshaled)
	if err != nil {
		t.Errorf("got error `Unmarshal`ing default config: %v, raw = %s", err, marshaled)
		t.FailNow()
	}
	if !reflect.DeepEqual(cfg, roundTripConfig) {
		t.Errorf("default config does not match after Unmarshal(Marshal()), %v != %v", cfg, roundTripConfig)
		t.FailNow()
	}
}

func TestUnmarshal(t *testing.T) {
	defaultConfig, err := Marshal(config.New())
	if err != nil {
		t.Errorf("Error setting up default config for test: %v", err)
		t.FailNow()
	}
	cases := []struct {
		Name           string
		Raw            []byte
		ExpectedConfig config.Any
		ExpectError    bool
	}{
		{
			Name:           "default config",
			Raw:            defaultConfig,
			ExpectedConfig: config.New(),
			ExpectError:    false,
		},
		{
			Name: "config with lifecycle",
			Raw: []byte(`kind: Config
apiVersion: kind.sigs.k8s.io/v1alpha1
nodeLifecycle:
  preKubeadm:
  - name: "pull an image"
    command: [ "docker", "pull", "ubuntu" ]
  - name: "pull another image"
    command: [ "docker", "pull", "debian" ]
    mustSucceed: true
`),
			ExpectedConfig: func() config.Any {
				cfg := &config.Config{
					NodeLifecycle: &config.NodeLifecycle{
						PreKubeadm: []config.LifecycleHook{
							{
								Name:    "pull an image",
								Command: []string{"docker", "pull", "ubuntu"},
							},
							{
								Name:        "pull another image",
								Command:     []string{"docker", "pull", "debian"},
								MustSucceed: true,
							},
						},
					},
				}
				cfg.ApplyDefaults()
				return cfg
			}(),
			ExpectError: false,
		},
		{
			Name:        "Invalid apiVersion 🤔",
			Raw:         []byte("kind: KindConfig\napiVersion: 🤔"),
			ExpectError: true,
		},
		{
			Name:        "generically invalid yaml",
			Raw:         []byte("\""),
			ExpectError: true,
		},
		{
			Name:        "invalid config yaml",
			Raw:         []byte("numNodes: too-many"),
			ExpectError: true,
		},
		{
			Name:        "nil input",
			Raw:         nil,
			ExpectError: true,
		},
	}
	for _, tc := range cases {
		cfg, err := Unmarshal(tc.Raw)
		if err != nil && !tc.ExpectError {
			t.Errorf(
				"case: '%s' got error `Unmarshal`ing and expected none: %v",
				tc.Name, err,
			)
		} else if err == nil && tc.ExpectError {
			t.Errorf(
				"case: '%s' got no error `Unmarshal`ing but expected one",
				tc.Name,
			)
		}
		if !reflect.DeepEqual(cfg, tc.ExpectedConfig) {
			t.Errorf(
				"case: '%s' `Unmarshal` result does not match expected: %v != %v",
				tc.Name, cfg, tc.ExpectedConfig,
			)
		}
	}
}

func TestUnmarshalDefaulting(t *testing.T) {
	// marshal an unset config
	emptyConfig, err := Marshal(&config.Config{})
	if err != nil {
		t.Errorf("Error setting up default config for test: %v", err)
		t.FailNow()
	}
	// create a config with defaulted values
	defaulted := &config.Config{}
	defaulted.ApplyDefaults()
	// unmarshal the unset config
	unmarshaledEmpty, err := Unmarshal(emptyConfig)
	if err != nil {
		t.Errorf("Error `Unmarshal`ing default config: %v", err)
		t.FailNow()
	}
	// verify that the unset config should match the default config
	if !reflect.DeepEqual(unmarshaledEmpty, defaulted) {
		t.Errorf(
			"defaulted config does not match unmarshaled empty config: %v != %v",
			defaulted, unmarshaledEmpty,
		)
		t.FailNow()
	}
}

// un-json.Marshal-able type
type unencodable <-chan int

// "implement" config.Any
func (u unencodable) ApplyDefaults()            {}
func (u unencodable) ToCurrent() *config.Config { return nil }
func (u unencodable) Validate() error           { return nil }
func (u unencodable) Kind() string              { return "bogus" }
func (u unencodable) APIVersion() string        { return "bogus" }

func TestMarshal(t *testing.T) {
	cases := []struct {
		Name        string
		Config      config.Any
		ExpectError bool
	}{
		{
			Name:        "nil config",
			Config:      nil,
			ExpectError: true,
		},
		{
			Name:        "wrong struct",
			Config:      make(unencodable),
			ExpectError: true,
		},
	}
	for _, tc := range cases {
		_, err := Marshal(tc.Config)
		if err != nil && !tc.ExpectError {
			t.Errorf("case: '%s' got error `Marshal`ing and expected none: %v", tc.Name, err)
		} else if err == nil && tc.ExpectError {
			t.Errorf("case: '%s' got no error `Marshal`ing but expected one", tc.Name)
		}
	}
}
