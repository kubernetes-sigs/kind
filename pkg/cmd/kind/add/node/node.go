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

// Package node implements the `add node` command
package node

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/internal/runtime"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	ClusterName string
	NodeName    string
	Role        string
	Image       string
	Retain      bool
}

// NewCommand returns a new cobra.Command for adding a node
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "node NODE_NAME",
		Short: "Add a node to an existing cluster",
		Long:  "Add a node to an existing cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.OverrideDefaultName(cmd.Flags())
			flags.NodeName = args[0]
			return runE(logger, streams, flags)
		},
	}
	cmd.Flags().StringVarP(
		&flags.ClusterName,
		"name",
		"n",
		cluster.DefaultName,
		"the cluster name",
	)
	cmd.Flags().StringVarP(
		&flags.Role,
		"role",
		"r",
		"worker",
		"the node role (worker or control-plane)",
	)
	cmd.Flags().StringVarP(
		&flags.Image,
		"image",
		"i",
		"",
		"the node image to use",
	)
	cmd.Flags().BoolVar(
		&flags.Retain,
		"retain",
		false,
		"retain the node container for debugging even if join fails",
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
		if c == flags.ClusterName {
			clusterExists = true
			break
		}
	}
	if !clusterExists {
		return fmt.Errorf("cluster %q does not exist", flags.ClusterName)
	}

	// Validate the role
	var role config.NodeRole
	switch flags.Role {
	case "worker":
		role = config.WorkerRole
	case "control-plane":
		role = config.ControlPlaneRole
	default:
		return fmt.Errorf("invalid role %q, must be one of: worker, control-plane", flags.Role)
	}

	// Create the node configuration with defaults
	nodeConfig := &config.Node{
		Role: role,
	}
	if flags.Image != "" {
		nodeConfig.Image = flags.Image
	}

	logger.V(0).Infof("Adding %s node %q to cluster %q ...", flags.Role, flags.NodeName, flags.ClusterName)

	// Add the node using the provider
	node, err := provider.AddNode(flags.ClusterName, flags.NodeName, nodeConfig, flags.Retain)
	if err != nil {
		return fmt.Errorf("failed to add node: %v", err)
	}

	logger.V(0).Infof("Node %s created and joined successfully", node.String())
	return nil
}
