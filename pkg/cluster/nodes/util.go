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

// GetControlPlaneEndpoint returns the control plane endpoint
// in case the cluster has an external load balancer it returns its ip,
// otherwise return the bootstrap node ip.
func GetControlPlaneEndpoint(allNodes []Node) (string, error) {
	node, err := ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return "", err
	}
	controlPlanePort := loadbalancer.ControlPlanePort
	// if there is no external load balancer use the bootstrap node
	if node == nil {
		node, err = BootstrapControlPlaneNode(allNodes)
		if err != nil {
			return "", err
		}
		controlPlanePort = kubeadm.APIServerPort
	}

	// get the IP and port for the load balancer
	controlPlaneIP, err := node.IP()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get IP for node: %s", node.Name())
	}

	return fmt.Sprintf("%s:%d", controlPlaneIP, controlPlanePort), nil
}
