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

// Package loadbalancer implements the load balancer configuration action
package loadbalancer

import (
	"fmt"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/haproxy"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

// Action implements and action for configuring and starting the
// external load balancer in front of the control-plane nodes.
type Action struct{}

// NewAction returns a new Action for configuring the load balancer
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// identify external load balancer node
	loadBalancerNode, err := nodes.ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return err
	}

	// if there's no loadbalancer we're done
	if loadBalancerNode == nil {
		return nil
	}

	// otherwise notify the user
	ctx.Status.Start("Starting the external load balancer ⚖️")
	defer ctx.Status.End(false)

	// collect info about the existing controlplane nodes
	var backendServers = map[string]string{}
	controlPlaneNodes, err := nodes.SelectNodesByRole(
		allNodes,
		constants.ControlPlaneNodeRoleValue,
	)
	if err != nil {
		return err
	}
	for _, n := range controlPlaneNodes {
		controlPlaneIP, err := n.IP()
		if err != nil {
			return errors.Wrapf(err, "failed to get IP for node %s", n.Name())
		}
		backendServers[n.Name()] = fmt.Sprintf("%s:%d", controlPlaneIP, kubeadm.APIServerPort)
	}

	// create haproxy config data
	haproxyConfig, err := haproxy.Config(&haproxy.ConfigData{
		ControlPlanePort: haproxy.ControlPlanePort,
		BackendServers:   backendServers,
	})
	if err != nil {
		return errors.Wrap(err, "failed to generate haproxy data")
	}

	// create haproxy config on the node
	if err := loadBalancerNode.WriteFile("/kind/haproxy.cfg", haproxyConfig); err != nil {
		// TODO: logging here
		return errors.Wrap(err, "failed to copy haproxy config to node")
	}

	// starts a docker container with HA proxy load balancer
	if err := loadBalancerNode.Command(
		"/bin/sh", "-c",
		fmt.Sprintf(
			"docker run -d -v /kind/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro --network host --restart always --name haproxy %s",
			haproxy.Image,
		),
	).Run(); err != nil {
		return errors.Wrap(err, "failed to start haproxy")
	}

	ctx.Status.End(true)
	return nil
}
