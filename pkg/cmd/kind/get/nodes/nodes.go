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

// Package nodes implements the `nodes` command
package nodes

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/internal/util/cli"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the list of nodes for a given cluster
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "nodes",
		Short: "lists existing kind nodes by their name",
		Long:  "lists existing kind nodes by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, logger, streams, flags)
		},
	}
	cmd.Flags().String(
		"name",
		"",
		fmt.Sprintf(`the cluster name. Overrides KIND_CLUSTER_NAME environment variable (default "%s")`, cluster.DefaultName),
	)
	return cmd
}

func runE(cmd *cobra.Command, logger log.Logger, streams cmd.IOStreams, flags *flagpole) error {
	// List nodes by cluster context name
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)
	flags.Name = cli.GetClusterNameFlags(cmd)
	n, err := provider.ListNodes(flags.Name)
	if err != nil {
		return err
	}
	for _, node := range n {
		fmt.Fprintln(streams.Out, node.String())
	}
	return nil
}
