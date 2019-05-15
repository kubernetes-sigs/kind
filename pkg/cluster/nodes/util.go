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

	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer"
)

// GetControlPlaneEndpoint returns the control plane endpoints for IPv4 and IPv6
// in case the cluster has an external load balancer in front of the control-plane nodes,
// otherwise return the bootstrap node IPs
func GetControlPlaneEndpoint(allNodes []Node) (string, string, error) {
	node, err := ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return "", "", err
	}
	controlPlanePort := loadbalancer.ControlPlanePort
	// if there is no external load balancer use the bootstrap node
	if node == nil {
		node, err = BootstrapControlPlaneNode(allNodes)
		if err != nil {
			return "", "", err
		}
		controlPlanePort = kubeadm.APIServerPort
	}

	// gets the control plane IP addresses
	controlPlaneIPv4, controlPlaneIPv6, err := node.IP()
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get IPs for node: %s", node.Name())
	}
	return fmt.Sprintf("%s:%d", controlPlaneIPv4, controlPlanePort), fmt.Sprintf("[%s]:%d", controlPlaneIPv6, controlPlanePort), nil
}
