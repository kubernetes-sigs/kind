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
	"os"
	"path/filepath"
	"regexp"
)

// ClusterLabelKey is applied to each "node" docker container for identification
const ClusterLabelKey = "io.k8s.test-infra.kind-cluster"

// similar to valid docker container names, but since we will prefix
// and suffix this name, we can relax it a little
// see Validate() for usage
var validNameRE = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// Config contains cluster options
type Config struct {
	// the cluster name
	Name string
	// the number of nodes (currently only one is supported)
	NumNodes int
	// TODO(bentheelder): fill this in
}

// NewConfig returns a new cluster config with name
func NewConfig(name string) Config {
	return Config{
		Name:     name,
		NumNodes: 1,
	}
}

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *Config) Validate() error {
	errs := []error{}
	if !validNameRE.MatchString(c.Name) {
		errs = append(errs, fmt.Errorf(
			"'%s' is not a valid cluster name, cluster names must match `%s`",
			c.Name, validNameRE.String(),
		))
	}
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

// internal helper used to identify the cluster containers based on config
func (c *Config) clusterLabel() string {
	return fmt.Sprintf("%s=%s", ClusterLabelKey, c.Name)
}

// ClusterName returns the Kubernetes cluster name based on the config
// currently this is .Name prefixed with "kind-"
func (c *Config) ClusterName() string {
	return fmt.Sprintf("kind-%s", c.Name)
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Config) KubeConfigPath() string {
	// TODO(bentheelder): Windows?
	// configDir matches the standard directory expected by kubectl etc
	configDir := filepath.Join(os.Getenv("HOME"), ".kube")
	// note that the file name however does not, we do not want to overwite
	// the standard config, though in the future we may (?) merge them
	fileName := fmt.Sprintf("kind-config-%s", c.Name)
	return filepath.Join(configDir, fileName)
}
