/*
Copyright 2018 The Kubernetes Authors.

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

// Package cluster implements the `delete` command
package cluster

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
)

type flags struct {
	Name   string
	Retain bool
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flags{}
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "cluster",
		Short: "Deletes a cluster",
		Long:  "Deletes a resource",
		Run: func(cmd *cobra.Command, args []string) {
			run(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", "1", "the cluster name")
	cmd.Flags().BoolVar(&flags.Retain, "retain", false, "whether retain the broken nodes for debugging")
	return cmd
}

func run(flags *flags, cmd *cobra.Command, args []string) {
	// TODO(bentheelder): make this more configurable
	ctx, err := cluster.NewContext(flags.Name, flags.Retain)
	if err != nil {
		log.Fatalf("Failed to create cluster context! %v", err)
	}
	err = ctx.Delete()
	if err != nil {
		log.Fatalf("Failed to delete cluster: %v", err)
	}
}
