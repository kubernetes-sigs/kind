/*
Copyright 2025 The Kubernetes Authors.

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

// Package node implements the `delete node` command
package node

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/internal/runtime"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Name     string
	NodeName string
}

// NewCommand returns a new cobra.Command for deleting a node
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "node [NODE_NAME]",
		Short: "Remove a node from an existing cluster",
		Long:  "Remove a node from an existing cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.OverrideDefaultName(cmd.Flags())
			flags.NodeName = args[0]
			return runE(logger, streams, flags)
		},
	}
	cmd.Flags().StringVarP(
		&flags.Name,
		"name",
		"n",
		cluster.DefaultName,
		"the cluster name",
	)
	return cmd
}

func runE(logger log.Logger, streams cmd.IOStreams, flags *flagpole) error {
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
		runtime.GetDefault(logger),
	)

	// Validate cluster exists
	clusters, err := provider.List()
	if err != nil {
		return fmt.Errorf("failed to list clusters: %v", err)
	}
	clusterExists := false
	for _, c := range clusters {
		if c == flags.Name {
			clusterExists = true
			break
		}
	}
	if !clusterExists {
		return fmt.Errorf("cluster %q does not exist", flags.Name)
	}

	// Validate node exists and check control plane safety
	clusterNodes, err := provider.ListNodes(flags.Name)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}
	var nodeToDelete nodes.Node
	nodeExists := false
	for _, n := range clusterNodes {
		if n.String() == flags.NodeName {
			nodeToDelete = n
			nodeExists = true
			break
		}
	}
	if !nodeExists {
		return fmt.Errorf("node %q does not exist in cluster %q", flags.NodeName, flags.Name)
	}

	// Check if this is a control plane node and if it's the last one
	nodeRole, err := nodeToDelete.Role()
	if err != nil {
		return fmt.Errorf("failed to determine node role: %v", err)
	}
	if nodeRole == constants.ControlPlaneNodeRoleValue {
		controlPlaneNodes, err := nodeutils.ControlPlaneNodes(clusterNodes)
		if err != nil {
			return fmt.Errorf("failed to list control plane nodes: %v", err)
		}
		if len(controlPlaneNodes) <= 1 {
			return fmt.Errorf("cannot delete the last control plane node")
		}
	}

	logger.V(0).Infof("Removing node %s from cluster %q ...", flags.NodeName, flags.Name)

	// Remove the node using the provider
	if err := provider.RemoveNode(flags.Name, flags.NodeName); err != nil {
		return fmt.Errorf("failed to remove node: %v", err)
	}

	logger.V(0).Infof("Node %s removed successfully", flags.NodeName)
	return nil
}
