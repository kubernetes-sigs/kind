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

// Package postcreationconfiguration implements the action for configuring cluster after cluster creation
// like labelling, tainting, etc the nodes.
package postcreationconfiguration

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// Action implements and action for configuring the cluster after cluster creation
type Action struct{}

// NewAction returns a new Action for configuring the cluster after cluster creation
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {

	// Label the control-plane and worker nodes as per the config file
	if err := labelNodes(ctx); err != nil {
		return err
	}

	return nil
}

// labelNodes labels the control-plane and worker nodes
// as per the labels provided in the config file
func labelNodes(ctx *actions.ActionContext) error {
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// Get one of the cluster nodes for the execution of labelling action/command.
	// The labelling action/command for a control-plane node will be executed by that respective node only. (label will be self-applied for the control-plane node)
	// The labelling action/command for a worker node will be executed by the below 'firstControlPlane' node.
	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	firstControlPlaneNode := controlPlanes[0]

	// Populate the list of control-plane node labels and the list of worker node labels respectively.
	// controlPlaneLabels is an array of maps (labels, read from config) associated with all the control-plane nodes.
	// workerLabels is an array of maps (labels, read from config) associated with all the worker nodes.
	controlPlaneLabels := []map[string]string{}
	workerLabels := []map[string]string{}
	for _, node := range ctx.Config.Nodes {
		if node.Role == config.ControlPlaneRole {
			controlPlaneLabels = append(controlPlaneLabels, node.Labels)
		} else if node.Role == config.WorkerRole {
			workerLabels = append(workerLabels, node.Labels)
		} else {
			continue
		}
	}

	// Label the control-plane and worker nodes accordingly.
	controlPlaneLabelsCounter := 0
	workerLabelsCounter := 0
	for _, node := range allNodes {
		nodeName := node.String()
		nodeRole, err := node.Role()
		if err != nil {
			return err
		}

		var currentNodeLabels map[string]string
		var cmdExecutorNode nodes.Node
		if nodeRole == string(config.ControlPlaneRole) {
			cmdExecutorNode = node
			currentNodeLabels = controlPlaneLabels[controlPlaneLabelsCounter]
			controlPlaneLabelsCounter++
		} else if nodeRole == string(config.WorkerRole) {
			cmdExecutorNode = firstControlPlaneNode
			currentNodeLabels = workerLabels[workerLabelsCounter]
			workerLabelsCounter++
		} else {
			continue
		}
		if len(currentNodeLabels) == 0 {
			continue
		}

		// Construct the `kubectl label` command as per the current labels for the current node in the loop.
		labelCommandArgsStr := fmt.Sprintf("--kubeconfig=/etc/kubernetes/admin.conf label node %s ", nodeName)
		for key, value := range currentNodeLabels {
			labelCommandArgsStr += fmt.Sprintf("%s=%s ", key, value)
		}
		labelCommandArgs := strings.Split(strings.TrimSpace(labelCommandArgsStr), " ")

		// Execute the constructed `kubectl load` command via the rightful node
		// If the labels are being applied to a control-plane node, then, they will be self-applied by that node
		// If the labels are being applied to a worker node, then, they will be applied by a control-plane node ('firstControlPlaneNode')
		if err := cmdExecutorNode.Command("kubectl", labelCommandArgs...).Run(); err != nil {
			return err
		}
	}

	return nil
}
