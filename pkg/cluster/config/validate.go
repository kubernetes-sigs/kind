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

package config

import (
	"fmt"

	"sigs.k8s.io/kind/pkg/util"
)

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *Config) Validate() error {
	errs := []error{}

	// All nodes in the config should be valid
	for i, n := range c.Nodes {
		if err := n.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("please fix invalid configuration for node %d: \n%v", i, err))
		}
	}

	if len(errs) > 0 {
		return util.NewErrors(errs)
	}
	return nil
}

// Validate returns a ConfigErrors with an entry for each problem
// with the Node, or nil if there are none
func (n *Node) Validate() error {
	errs := []error{}

	// validate node role should be one of the expected values
	switch n.Role {
	case ControlPlaneRole,
		WorkerRole,
		ExternalEtcdRole,
		ExternalLoadBalancerRole:
	default:
		errs = append(errs, fmt.Errorf("role is a required field"))
	}

	// image should be defined
	if n.Image == "" {
		errs = append(errs, fmt.Errorf("image is a required field"))
	}

	// replicas >= 0
	if n.Replicas != nil && int32(*n.Replicas) < 0 {
		errs = append(errs, fmt.Errorf("replicas number should not be a negative number"))
	}

	if len(errs) > 0 {
		return util.NewErrors(errs)
	}

	return nil
}
