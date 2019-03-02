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
	"sort"
	"time"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// provisioning order for nodes by role
var defaultRoleOrder = []string{
	constants.ExternalLoadBalancerNodeRoleValue,
	constants.ExternalEtcdNodeRoleValue,
	constants.ControlPlaneNodeRoleValue,
	constants.WorkerNodeRoleValue,
}

// sorts nodes for provisioning
func sortNodes(nodes []config.Node, roleOrder []string) {
	roleToOrder := makeRoleToOrder(roleOrder)
	sort.SliceStable(nodes, func(i, j int) bool {
		return roleToOrder(string(nodes[i].Role)) < roleToOrder(string(nodes[j].Role))
	})
}

// helper to convert an ordered slice of roles to a mapping of provisioning
// role to provisioning order
func makeRoleToOrder(roleOrder []string) func(string) int {
	orderMap := make(map[string]int)
	for i, role := range roleOrder {
		orderMap[role] = i
	}
	return func(role string) int {
		p, ok := orderMap[role]
		if !ok {
			return 10000
		}
		return p
	}
}

// TODO(bentheelder): eliminate this when we have v1alpha3
func convertReplicas(nodes []config.Node) []config.Node {
	out := []config.Node{}
	for _, node := range nodes {
		replicas := int32(1)
		if node.Replicas != nil {
			replicas = *node.Replicas
		}
		for i := int32(0); i < replicas; i++ {
			outNode := node.DeepCopy()
			outNode.Replicas = nil
			out = append(out, *outNode)
		}
	}
	return out
}

// provisionNodes takes care of creating all the containers
// that will host `kind` nodes
func provisionNodes(
	status *logutil.Status, cfg *config.Config, clusterName, clusterLabel string,
) (nodesByName map[string]*nodes.Node, err error) {
	nodesByName = map[string]*nodes.Node{}

	// convert replicas to normal nodes
	// TODO(bentheelder): eliminate this when we have v1alpha3
	configNodes := convertReplicas(cfg.Nodes)
	// TODO(bentheelder): allow overriding defaultRoleOrder
	sortNodes(configNodes, defaultRoleOrder)

	nameNode := makeNodeNamer(clusterName)

	// provision all nodes in the config
	// TODO(bentheelder): handle implicit nodes as well
	for _, configNode := range configNodes {
		// name the node
		name := nameNode(string(configNode.Role))

		// create the node into a container (docker run, but it is paused, see createNode)
		status.Start(fmt.Sprintf("[%s] Creating node container 📦", name))
		var node *nodes.Node
		// TODO(bentheelder): decouple from config objects further
		switch string(configNode.Role) {
		case constants.ExternalLoadBalancerNodeRoleValue:
			node, err = nodes.CreateExternalLoadBalancerNode(name, configNode.Image, clusterLabel)
		case constants.ControlPlaneNodeRoleValue:
			node, err = nodes.CreateControlPlaneNode(name, configNode.Image, clusterLabel, configNode.ExtraMounts)
		case constants.WorkerNodeRoleValue:
			node, err = nodes.CreateWorkerNode(name, configNode.Image, clusterLabel, configNode.ExtraMounts)
		}
		if err != nil {
			return nodesByName, err
		}
		nodesByName[name] = node

		status.Start(fmt.Sprintf("[%s] Fixing mounts 🗻", name))
		// we need to change a few mounts once we have the container
		// we'd do this ahead of time if we could, but --privileged implies things
		// that don't seem to be configurable, and we need that flag
		if err := node.FixMounts(); err != nil {
			// TODO(bentheelder): logging here
			return nodesByName, err
		}

		status.Start(fmt.Sprintf("[%s] Configuring proxy 🐋", name))
		if err := node.SetProxy(); err != nil {
			// TODO: logging here
			return nodesByName, errors.Wrapf(err, "failed to set proxy for %s", name)
		}

		status.Start(fmt.Sprintf("[%s] Starting systemd 🖥", name))
		// signal the node container entrypoint to continue booting into systemd
		if err := node.SignalStart(); err != nil {
			// TODO(bentheelder): logging here
			return nodesByName, err
		}

		status.Start(fmt.Sprintf("[%s] Waiting for docker to be ready 🐋", name))
		// wait for docker to be ready
		if !node.WaitForDocker(time.Now().Add(time.Second * 30)) {
			// TODO(bentheelder): logging here
			return nodesByName, errors.New("timed out waiting for docker to be ready on node")
		}

		// load the docker image artifacts into the docker daemon
		status.Start(fmt.Sprintf("[%s] Pre-loading images 🐋", name))
		node.LoadImages()

	}

	return nodesByName, nil
}

func makeNodeNamer(clusterName string) func(string) string {
	counter := make(map[string]int)
	return func(role string) string {
		count := 0
		suffix := ""
		if v, ok := counter[role]; ok {
			count += v
			suffix = fmt.Sprintf("%d", count)
		}
		counter[role] = count
		return fmt.Sprintf("%s-%s%s", clusterName, role, suffix)
	}
}
