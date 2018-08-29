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

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

// CreateConfig contains cluster creation config
type CreateConfig struct {
	// NumNodes is the number of nodes to create (currently only one is supported)
	NumNodes int `json:"numNodes"`
	// KubeadmConfigTemplate allows overriding the default template in
	// cluster/kubeadm
	KubeadmConfigTemplate string `json:"kubeadmConfigTemplate"`
}

// NewCreateConfig returns a new default CreateConfig
func NewCreateConfig() *CreateConfig {
	return &CreateConfig{
		NumNodes: 1,
	}
}

// LoadCreateConfig reads the file at path and attempts to load it as
// a yaml encoding of CreateConfig, falling back to json if this fails.
// It returns an error if reading the files fails, or if both yaml and json fail
// If path is "" then a default config is returned instead
func LoadCreateConfig(path string) (config *CreateConfig, err error) {
	if path == "" {
		return NewCreateConfig(), nil
	}
	// read in file
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// first try yaml
	config = &CreateConfig{}
	yamlErr := yaml.Unmarshal(contents, config)
	if yamlErr == nil {
		return config, nil
	}
	// then try json
	config = &CreateConfig{}
	jsonErr := json.Unmarshal(contents, config)
	if jsonErr == nil {
		return config, nil
	}
	return nil, fmt.Errorf("could not read as yaml: %v or json: %v", yamlErr, jsonErr)
}

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *CreateConfig) Validate() error {
	errs := []error{}
	// TODO(bentheelder): support multiple nodes
	if c.NumNodes != 1 {
		errs = append(errs, fmt.Errorf(
			"%d nodes requested but only clusters with one node are supported currently",
			c.NumNodes,
		))
	}
	if len(errs) > 0 {
		return ConfigErrors{errs}
	}
	return nil
}

// ConfigErrors implements error and contains all config errors
// This is returned by Config.Validate
type ConfigErrors struct {
	errors []error
}

// assert ConfigErrors implements error
var _ error = &ConfigErrors{}

func (c ConfigErrors) Error() string {
	var buff bytes.Buffer
	for _, err := range c.errors {
		buff.WriteString(err.Error())
		buff.WriteRune('\n')
	}
	return buff.String()
}

// Errors returns the slice of errors contained by ConfigErrors
func (c ConfigErrors) Errors() []error {
	return c.errors
}
