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
		Short: "Lists existing kind nodes by their name",
		Long:  "Lists existing kind nodes by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, streams, flags)
		},
	}
	cmd.Flags().StringVar(
		&flags.Name,
		"name",
		cluster.DefaultName,
		"the cluster context name",
	)
	return cmd
}

func runE(logger log.Logger, streams cmd.IOStreams, flags *flagpole) error {
	// List nodes by cluster context name
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)
	n, err := provider.ListNodes(flags.Name)
	if err != nil {
		return err
	}
	if len(n) == 0 {
		logger.V(0).Infof("No kind nodes found for cluster %q.", flags.Name)
		return nil
	}
	for _, node := range n {
		fmt.Fprintln(streams.Out, node.String())
	}
	return nil
}
