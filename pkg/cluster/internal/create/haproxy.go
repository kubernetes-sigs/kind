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

package create

import (
	"fmt"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/haproxy"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
)

// HAProxyAction implements action for configuring and starting the
// external load balancer in front of the control-plane nodes.
type HAProxyAction struct{}

func init() {
	registerAction("haproxy", NewHAProxyAction)
}

// NewHAProxyAction returns a new HAProxyAction
func NewHAProxyAction() Action {
	return &HAProxyAction{}
}

// Tasks returns the list of action tasks
func (b *HAProxyAction) Tasks() []Task {
	return []Task{
		{
			Description: "Starting the external load balancer ⛵",
			TargetNodes: selectExternalLoadBalancerNode,
			Run:         runHAProxy,
		},
	}
}

// runKubeadmJoin executes haproxy
func runHAProxy(ec *execContext, configNode *NodeReplica) error {
	// collects info about the existing controlplane nodes
	var backendServers = map[string]string{}
	for _, n := range ec.ControlPlanes() {
		// gets the handle for the control plane node
		controlPlaneHandle, ok := ec.NodeFor(n)
		if !ok {
			return errors.Errorf("unable to get the handle for operating on node: %s", n.Name)
		}

		// gets the IP of the control plane node
		controlPlaneIP, err := controlPlaneHandle.IP()
		if err != nil {
			return errors.Wrapf(err, "failed to get IP for node %s", n.Name)
		}

		backendServers[n.Name] = fmt.Sprintf("%s:%d", controlPlaneIP, kubeadm.APIServerPort)
	}

	// create haproxy config data
	haproxyConfig, err := haproxy.Config(&haproxy.ConfigData{
		ControlPlanePort: haproxy.ControlPlanePort,
		BackendServers:   backendServers,
	},
	)

	if err != nil {
		return errors.Wrap(err, "failed to generate haproxy data")
	}

	// get the target node for this task (the load balancer node)
	node, ok := ec.NodeFor(configNode)
	if !ok {
		return errors.Errorf("unable to get the handle for operating on node: %s", configNode.Name)
	}

	// create haproxy config on the node
	if err := node.WriteFile("/kind/haproxy.cfg", haproxyConfig); err != nil {
		// TODO: logging here
		return errors.Wrap(err, "failed to copy haproxy config to node")
	}

	// starts a docker container with HA proxy load balancer
	if err := node.Command(
		"/bin/sh", "-c",
		fmt.Sprintf("docker run -d -v /kind/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro --network host --restart always %s", haproxy.Image),
	).Run(); err != nil {
		return errors.Wrap(err, "failed to start haproxy")
	}

	return nil
}
