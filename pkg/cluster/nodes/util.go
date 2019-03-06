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

package nodes

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/internal/haproxy"
)

// GetControlPlaneEndpoint returns the control plane endpoint in case the
// cluster has an external load balancer in front of the control-plane nodes,
// otherwise return an empty string.
func GetControlPlaneEndpoint(allNodes []Node) (string, error) {
	node, err := ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return "", err
	}
	// if there is no external load balancer
	if node == nil {
		return "", nil
	}
	// gets the IP of the load balancer
	loadBalancerIP, err := node.IP()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get IP for node: %s", node.Name())
	}
	return fmt.Sprintf("%s:%d", loadBalancerIP, haproxy.ControlPlanePort), nil
}
