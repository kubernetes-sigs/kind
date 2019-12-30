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

// Package registry implements the image registry initialization
package registry

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
)

// Action implements action for updating the nodes /etc/hosts
// with the kind-registry node IP address.
type Action struct{}

// NewAction returns a new action for updating the nodes /etc/hosts
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// identify image registry node
	registry, err := nodeutils.ImageRegistryNode(allNodes)
	if err != nil {
		return err
	}

	// get the node ip address
	registryIP, registryIPv6, err := registry.IP()
	if err != nil {
		return errors.Wrap(err, "failed to get IP for node")
	}

	// configure the right protocol addresses
	if ctx.Config.Networking.IPFamily == "ipv6" {
		registryIP = registryIPv6
	}

	// configure the nodes
	if err := configureNodes(ctx, registryIP, allNodes); err != nil {
		return err
	}

	return nil
}

func configureNodes(
	ctx *actions.ActionContext,
	registryIP string,
	nodes []nodes.Node,
) error {
	ctx.Status.Start("Configuring Image registry 📜")
	defer ctx.Status.End(false)

	// update the nodes concurrently
	fns := []func() error{}
	for _, node := range nodes {
		node := node // capture loop variable
		if node.String() != "kind-registry" {
			fns = append(fns, func() error {
				return updateHosts(ctx.Logger, registryIP, node)
			})
		}
	}
	if err := errors.UntilErrorConcurrent(fns); err != nil {
		return err
	}

	ctx.Status.End(true)
	return nil
}

// updateHosts executes command to add kind-registry to /etc/hosts on the node.
func updateHosts(logger log.Logger, registryIP string, node nodes.Node) error {
	// update /etc/hosts
	cmd := node.Command(
		"sh", "-c",
		fmt.Sprintf("echo %s kind-registry >> /etc/hosts", registryIP),
	)
	lines, err := exec.CombinedOutputLines(cmd)
	logger.V(3).Info(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to update /etc/hosts")
	}

	return nil
}
