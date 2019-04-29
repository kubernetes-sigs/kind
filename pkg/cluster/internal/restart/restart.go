/*
Copyright 2019 The Kubernetes Authors.

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

package restart

import (
	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/context"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/loadbalancer"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

// Cluster restarts the cluster identified by ctx
func Cluster(c *context.Context) error {
	allNodes, err := c.ListNodes()
	if err != nil {
		return errors.Wrap(err, "error listing nodes")
	}

	// restart all of cluster's nodes.
	err = nodes.Restart(allNodes)
	if err != nil {
		return errors.Wrap(err, "error restarting nodes")
	}

	// after we execute docker restart, the IP inside the container may be changed.
	// we should change config.
	loadBalancerNode, err := nodes.ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return err
	}

	// if there is no external load balancer
	if loadBalancerNode == nil {
		return nil
	}

	err = loadbalancer.ConfigHAProxy(loadBalancerNode, allNodes)
	if err != nil {
		return err
	}

	return nil
}
