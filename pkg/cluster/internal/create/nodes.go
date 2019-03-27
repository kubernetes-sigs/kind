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
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/container/cri"
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

// returns a deep copy of a slice of config nodes
func copyConfigNodes(toCopy []config.Node) []config.Node {
	out := make([]config.Node, len(toCopy))
	for i, node := range toCopy {
		out[i] = *node.DeepCopy()
	}
	return out
}

// provisionNodes takes care of creating all the containers
// that will host `kind` nodes
func provisionNodes(
	status *logutil.Status, cfg *config.Cluster, clusterName, clusterLabel string,
) error {
	defer status.End(false)

	if err := createNodeContainers(status, cfg, clusterName, clusterLabel); err != nil {
		return err
	}

	status.End(true)
	return nil
}

func createNodeContainers(
	status *logutil.Status, cfg *config.Cluster, clusterName, clusterLabel string,
) error {
	defer status.End(false)

	// compute the desired nodes, and inform the user that we are setting them up
	desiredNodes := nodesToCreate(cfg, clusterName)
	status.Start("Preparing nodes " + strings.Repeat("📦", len(desiredNodes)))

	// create all of the node containers, concurrently
	errChan := make(chan error, len(desiredNodes))
	var wg sync.WaitGroup
	for _, desiredNode := range desiredNodes {
		desiredNode := desiredNode // capture loop variable
		wg.Add(1)
		go func() {
			defer wg.Done()
			// create the node into a container (docker run, but it is paused, see createNode)
			node, err := desiredNode.Create(clusterLabel)
			if err != nil {
				errChan <- err
				return
			}
			err = fixupNode(node)
			if err != nil {
				errChan <- err
				return
			}
			errChan <- nil
		}()
	}

	// wait for all node creation to be done before closing
	go func() {
		defer close(errChan)
		wg.Wait()
	}()

	// return the first error encountered if any
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	status.End(true)
	return nil
}

func fixupNode(node *nodes.Node) error {
	// we need to change a few mounts once we have the container
	// we'd do this ahead of time if we could, but --privileged implies things
	// that don't seem to be configurable, and we need that flag
	if err := node.FixMounts(); err != nil {
		// TODO(bentheelder): logging here
		return err
	}

	if nodes.NeedProxy() {
		if err := node.SetProxy(); err != nil {
			// TODO: logging here
			return errors.Wrapf(err, "failed to set proxy for node %s", node.Name())
		}
	}

	// signal the node container entrypoint to continue booting into systemd
	if err := node.SignalStart(); err != nil {
		// TODO(bentheelder): logging here
		return err
	}

	// wait for docker to be ready
	if !node.WaitForDocker(time.Now().Add(time.Second * 30)) {
		// TODO(bentheelder): logging here
		return errors.Errorf("timed out waiting for docker to be ready on node %s", node.Name())
	}

	// load the docker image artifacts into the docker daemon
	node.LoadImages()

	return nil
}

// nodeSpec describes a node to create purely from the container aspect
// this does not inlude eg starting kubernetes (see actions for that)
type nodeSpec struct {
	Name        string
	Role        string
	Image       string
	ExtraMounts []cri.Mount
	// TODO(bentheelder): replace with a cri.PortMapping when we have that
	APIServerPort    int32
	APIServerAddress string
}

func nodesToCreate(cfg *config.Cluster, clusterName string) []nodeSpec {
	desiredNodes := []nodeSpec{}

	// nodes are named based on the cluster name and their role, with a counter
	nameNode := makeNodeNamer(clusterName)

	// copy and sort config nodes
	// TODO(bentheelder): allow overriding defaultRoleOrder
	configNodes := copyConfigNodes(cfg.Nodes)
	sortNodes(configNodes, defaultRoleOrder)

	// determine if we are HA, and what the LB will need to know for that
	controlPlanes := 0
	controlPlaneImage := "" // the control plane LB will use this for now
	for _, configNode := range configNodes {
		role := string(configNode.Role)
		if role == constants.ControlPlaneNodeRoleValue {
			controlPlanes++
			// TODO(bentheelder): instead of keeping the first image, we
			// should have a config field that controls this image, and use
			// that with defaulting
			if controlPlaneImage == "" {
				controlPlaneImage = configNode.Image
			}
		}
	}
	isHA := controlPlanes > 1

	// add all of the config nodes as desired nodes
	for _, configNode := range configNodes {
		role := string(configNode.Role)
		apiServerPort := cfg.Networking.APIServerPort
		apiServerAddress := cfg.Networking.APIServerAddress
		// only the external LB should reflect the port if we have
		// multiple control planes
		if isHA && role != constants.ExternalLoadBalancerNodeRoleValue {
			apiServerPort = 0              // replaced with a random port
			apiServerAddress = "127.0.0.1" // only the LB needs to be non-local
		}
		desiredNodes = append(desiredNodes, nodeSpec{
			Name:             nameNode(role),
			Image:            configNode.Image,
			Role:             role,
			ExtraMounts:      configNode.ExtraMounts,
			APIServerAddress: apiServerAddress,
			APIServerPort:    apiServerPort,
		})
	}

	// add an external load balancer if there are multiple control planes
	if controlPlanes > 1 {
		role := constants.ExternalLoadBalancerNodeRoleValue
		desiredNodes = append(desiredNodes, nodeSpec{
			Name:        nameNode(role),
			Image:       controlPlaneImage, // TODO(bentheelder): get from config instead
			Role:        role,
			ExtraMounts: []cri.Mount{},
		})
	}

	return desiredNodes
}

// TODO(bentheelder): remove network in favor of []cri.PortMapping when that is in
func (d *nodeSpec) Create(clusterLabel string) (node *nodes.Node, err error) {
	// create the node into a container (docker run, but it is paused, see createNode)
	// TODO(bentheelder): decouple from config objects further
	switch d.Role {
	case constants.ExternalLoadBalancerNodeRoleValue:
		node, err = nodes.CreateExternalLoadBalancerNode(d.Name, d.Image, clusterLabel, d.APIServerAddress, d.APIServerPort)
	case constants.ControlPlaneNodeRoleValue:
		node, err = nodes.CreateControlPlaneNode(d.Name, d.Image, clusterLabel, d.APIServerAddress, d.APIServerPort, d.ExtraMounts)
	case constants.WorkerNodeRoleValue:
		node, err = nodes.CreateWorkerNode(d.Name, d.Image, clusterLabel, d.ExtraMounts)
	default:
		return nil, errors.Errorf("unknown node role: %s", d.Role)
	}
	return node, err
}

// makeNodeNamer returns a func(role string)(nodeName string)
// used to name nodes based on their role and the clusterName
func makeNodeNamer(clusterName string) func(string) string {
	counter := make(map[string]int)
	return func(role string) string {
		count := 1
		suffix := ""
		if v, ok := counter[role]; ok {
			count += v
			suffix = fmt.Sprintf("%d", count)
		}
		counter[role] = count
		return fmt.Sprintf("%s-%s%s", clusterName, role, suffix)
	}
}
