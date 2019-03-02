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
	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/util"
)

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *Config) Validate() error {
	errs := []error{}

	numByRole := make(map[NodeRole]int)
	// All nodes in the config should be valid
	for i, n := range c.Nodes {
		// update role count
		if num, ok := numByRole[n.Role]; ok {
			numByRole[n.Role] = 1 + num
		} else {
			numByRole[n.Role] = 1
		}
		// validate the node
		if err := n.Validate(); err != nil {
			errs = append(errs, errors.Errorf("invalid configuration for node %d: %v", i, err))
		}
	}

	// there must be at least one control plane node
	numControlPlane, anyControlPlane := numByRole[ControlPlaneRole]
	if !anyControlPlane || numControlPlane < 1 {
		errs = append(errs, errors.Errorf("must have at least one %s node", string(ControlPlaneRole)))
	}

	// there may not be more than one load balancer
	numLoadBlancer, _ := numByRole[ExternalLoadBalancerRole]
	if numLoadBlancer > 1 {
		errs = append(errs, errors.Errorf("only one %s node is supported", string(ExternalLoadBalancerRole)))
	}

	// there must be a load balancer if there are multiple control planes
	if numControlPlane > 1 && numLoadBlancer != 1 {
		errs = append(errs, errors.Errorf("%d > 1 %s nodes requires a %s node", numControlPlane, string(ControlPlaneRole), string(ExternalLoadBalancerRole)))
	}

	// external-etcd is not actually supported yet
	numExternalEtcd, _ := numByRole[ExternalEtcdRole]
	if numExternalEtcd > 0 {
		errs = append(errs, errors.Errorf("multi node support is still a work in progress, currently %s node is not supported", string(ExternalEtcdRole)))
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
		errs = append(errs, errors.Errorf("%q is not a valid node role", n.Role))
	}

	// image should be defined
	if n.Image == "" {
		errs = append(errs, errors.New("image is a required field"))
	}

	// replicas >= 0
	if n.Replicas != nil && int32(*n.Replicas) < 0 {
		errs = append(errs, errors.New("replicas number should not be a negative number"))
	}

	if len(errs) > 0 {
		return util.NewErrors(errs)
	}

	return nil
}
