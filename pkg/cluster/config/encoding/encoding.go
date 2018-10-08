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

// Package encoding implements apiVersion aware functionality to
// Marshal / Unmarshal / Load Config
package encoding

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"

	"sigs.k8s.io/kind/pkg/cluster/config"
)

// Load reads the file at path and attempts to load it as a yaml Config
// after detecting the apiVersion in the file
// (or defaulting to the current version if none is specified)
// If path == "" then the default config for the current version is returned
func Load(path string) (config.Any, error) {
	if path == "" {
		return config.New(), nil
	}
	// read in file
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// read in some config version
	// TODO(bentheelder): we do not use or respect `kind:` at all
	// possibly we should require something like `kind: "Config"`
	cfg, err := Unmarshal(contents)
	if err != nil {
		return nil, err
	}
	return cfg, err
}

// LoadCurrent is equivalent to Load followed by cfg.ToCurrent()
func LoadCurrent(path string) (*config.Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}
	return cfg.ToCurrent(), nil
}

// used to sniff a config for it's api version
type configOnlyVersion struct {
	APIVersion string `json:"apiVersion,omitempty"`
}

// helper to sniff and validate apiVersion
func detectVersion(raw []byte) (version string, err error) {
	c := configOnlyVersion{}
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return "", err
	}
	switch c.APIVersion {
	// default to the current api version if unspecified, or explicitly specified
	case config.SchemeGroupVersion.String(), "":
		return config.SchemeGroupVersion.String(), nil
	}
	return "", fmt.Errorf("invalid version: %v", c.APIVersion)
}

// Unmarshal is an apiVersion aware yaml.Unmarshall for config.Any
func Unmarshal(raw []byte) (config.Any, error) {
	if raw == nil {
		return nil, fmt.Errorf("nil input")
	}
	// sniff and validate version
	version, err := detectVersion(raw)
	if err != nil {
		return nil, err
	}
	// load version
	var cfg config.Any
	switch version {
	case config.SchemeGroupVersion.String():
		cfg = &config.Config{}
	}
	err = yaml.Unmarshal(raw, cfg)
	if err != nil {
		return nil, err
	}
	// apply defaults before returning
	cfg.ApplyDefaults()
	return cfg, nil
}

// used by `Marshal` to encode the config header
type configHeader struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

// Marshal marshals any config with kind and apiVersion header
func Marshal(cfg config.Any) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil input")
	}
	var buff bytes.Buffer
	// write kind, apiVersion header
	b, err := yaml.Marshal(configHeader{
		Kind:       cfg.Kind(),
		APIVersion: cfg.APIVersion(),
	})
	if err != nil {
		return nil, err
	}
	// NOTE: buff.Write can only fail with a panic if it cannot allocate
	buff.Write(b)
	// write actual config contents
	b, err = yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	// don't write a `{}` when the config has no set fields
	if !bytes.Equal(b, []byte("{}\n")) {
		buff.Write(b)
	}
	return buff.Bytes(), nil
}
