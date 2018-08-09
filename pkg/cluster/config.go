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
	"fmt"
	"regexp"
)

// ClusterLabelKey is applied to each "node" docker container for identification
const ClusterLabelKey = "io.k8s.test-infra.kind-cluster"

// similar to valid docker container names, but since we will prefix
// and suffix this name, we can relax it a little
// see Validate() for usage
var validClusterName = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// Config contains cluster options
type Config struct {
	// the cluster name
	Name string
	// TODO(bentheelder): fill this in
}

// NewConfig returns a new cluster config with name
func NewConfig(name string) Config {
	return Config{
		Name: name,
	}
}

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *Config) Validate() error {
	errs := []error{}
	if !validClusterName.MatchString(c.Name) {
		errs = append(errs, fmt.Errorf(
			"'%s' is not a valid cluster name, cluster names must match `%s`",
			c.Name, validClusterName.String(),
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

// internal helper used to identify the cluster containers based on config
func (c *Config) clusterLabel() string {
	return fmt.Sprintf("%s=%s", ClusterLabelKey, c.Name)
}
