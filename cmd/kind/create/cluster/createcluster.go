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

// Package cluster implements the `create cluster` command
package cluster

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/create"
	"sigs.k8s.io/kind/pkg/util"
)

type flagpole struct {
	Name      string
	Config    string
	ImageName string
	Retain    bool
	Wait      time.Duration
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a local Kubernetes cluster",
		Long:  "Creates a local Kubernetes cluster using Docker container 'nodes'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "cluster context name")
	cmd.Flags().StringVar(&flags.Config, "config", "", "path to a kind config file")
	cmd.Flags().StringVar(&flags.ImageName, "image", "", "node docker image to use for booting the cluster")
	cmd.Flags().BoolVar(&flags.Retain, "retain", false, "retain nodes for debugging when cluster creation fails")
	cmd.Flags().DurationVar(&flags.Wait, "wait", time.Duration(0), "Wait for control plane node to be ready (default 0s)")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// Check if the cluster name already exists
	known, err := cluster.IsKnown(flags.Name)
	if err != nil {
		return err
	}
	if known {
		return errors.Errorf("a cluster with the name %q already exists", flags.Name)
	}

	// create a cluster context and create the cluster
	ctx := cluster.NewContext(flags.Name)
	fmt.Printf("Creating cluster %q ...\n", flags.Name)
	if err = ctx.Create(
		create.WithConfigFile(flags.Config),
		create.WithNodeImage(flags.ImageName),
		create.Retain(flags.Retain),
		create.WaitForReady(flags.Wait),
	); err != nil {
		if utilErrors, ok := err.(util.Errors); ok {
			for _, problem := range utilErrors.Errors() {
				log.Error(problem)
			}
			return errors.New("aborting due to invalid configuration")
		}
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}
