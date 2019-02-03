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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	clusternodes "sigs.k8s.io/kind/pkg/cluster/nodes"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the list of nodes for a given cluster
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "nodes",
		Short: "lists existing kind nodes by their name",
		Long:  "lists existing kind nodes by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
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

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// List nodes by cluster context name
	n, err := clusternodes.ListByCluster()
	if err != nil {
		return err
	}
	nodes, known := n[flags.Name]
	if !known {
		return errors.Errorf("unknown cluster %q", flags.Name)
	}
	for _, node := range nodes {
		fmt.Println(node.String())
	}
	return nil
}
