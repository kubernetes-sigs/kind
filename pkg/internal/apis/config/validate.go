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
	"strings"

	utilnet "k8s.io/utils/net"
	"sigs.k8s.io/kind/pkg/errors"
)

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *Cluster) Validate() error {
	errs := []error{}

	// the api server port only needs checking if we aren't picking a random one
	// at runtime
	if c.Networking.APIServerPort != 0 {
		// validate api server listen port
		if err := validatePort(c.Networking.APIServerPort); err != nil {
			errs = append(errs, errors.Wrapf(err, "invalid apiServerPort"))
		}
	}

	isDualStack := c.Networking.IPFamily == "DualStack"
	// podSubnet should be a valid CIDR
	if err := validateSubnets(c.Networking.PodSubnet, isDualStack); err != nil {
		errs = append(errs, errors.Errorf("invalid pod subnet %v", err))
	}

	// serviceSubnet should be a valid CIDR
	if err := validateSubnets(c.Networking.ServiceSubnet, isDualStack); err != nil {
		errs = append(errs, errors.Errorf("invalid service subnet %v", err))
	}

	// validate nodes
	numByRole := make(map[NodeRole]int32)
	// All nodes in the config should be valid
	for i, n := range c.Nodes {
		// validate the node
		if err := n.Validate(); err != nil {
			errs = append(errs, errors.Errorf("invalid configuration for node %d: %v", i, err))
		}
		// update role count
		if num, ok := numByRole[n.Role]; ok {
			numByRole[n.Role] = 1 + num
		} else {
			numByRole[n.Role] = 1
		}
	}

	// there must be at least one control plane node
	numControlPlane, anyControlPlane := numByRole[ControlPlaneRole]
	if !anyControlPlane || numControlPlane < 1 {
		errs = append(errs, errors.Errorf("must have at least one %s node", string(ControlPlaneRole)))
	}

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
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
		WorkerRole:
	default:
		errs = append(errs, errors.Errorf("%q is not a valid node role", n.Role))
	}

	// image should be defined
	if n.Image == "" {
		errs = append(errs, errors.New("image is a required field"))
	}

	// validate extra port forwards
	for _, mapping := range n.ExtraPortMappings {
		if err := validatePort(mapping.HostPort); err != nil {
			errs = append(errs, errors.Wrapf(err, "invalid hostPort"))
		}
		if err := validatePort(mapping.ContainerPort); err != nil {
			errs = append(errs, errors.Wrapf(err, "invalid containerPort"))
		}
	}

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}

	return nil
}

func validatePort(port int32) error {
	if port < 0 || port > 65535 {
		return errors.Errorf("invalid port number: %d", port)
	}
	return nil
}

func validateSubnets(subnetStr string, dualstack bool) error {

	subnets, err := utilnet.ParseCIDRs(strings.Split(subnetStr, ","))
	if err != nil {
		return errors.Wrapf(err, "invalid subnet")
	}
	if dualstack {
		areDualStackCIDRs, err := utilnet.IsDualStackCIDRs(subnets)
		if err != nil {
			return errors.Wrapf(err, "invalid DualStack subnets")
		} else if !areDualStackCIDRs {
			return errors.Errorf("expected at least one IP from each family (v4 or v6) for DualStack networking: %q", subnetStr)
		}
	} else if len(subnets) != 1 {
		return errors.New("only one subnet allows")
	}
	return nil
}
