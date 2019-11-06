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

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Name       string
	Config     string
	ImageName  string
	Retain     bool
	Wait       time.Duration
	Kubeconfig string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a local Kubernetes cluster",
		Long:  "Creates a local Kubernetes cluster using Docker container 'nodes'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "cluster context name")
	cmd.Flags().StringVar(&flags.Config, "config", "", "path to a kind config file")
	cmd.Flags().StringVar(&flags.ImageName, "image", "", "node docker image to use for booting the cluster")
	cmd.Flags().BoolVar(&flags.Retain, "retain", false, "retain nodes for debugging when cluster creation fails")
	cmd.Flags().DurationVar(&flags.Wait, "wait", time.Duration(0), "Wait for control plane node to be ready (default 0s)")
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", "", "sets kubeconfig path instead of $KUBECONFIG or $HOME/.kube/config")
	return cmd
}

func runE(logger log.Logger, flags *flagpole) error {
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)

	// Check if the cluster name already exists
	n, err := provider.ListNodes(flags.Name)
	if err != nil {
		return err
	}
	if len(n) != 0 {
		return fmt.Errorf("a cluster with the name %q already exists", flags.Name)
	}

	// create the cluster
	logger.V(0).Infof("Creating cluster %q ...\n", flags.Name)
	if err = provider.Create(
		flags.Name,
		cluster.CreateWithConfigFile(flags.Config),
		cluster.CreateWithNodeImage(flags.ImageName),
		cluster.CreateWithRetain(flags.Retain),
		cluster.CreateWithWaitForReady(flags.Wait),
		cluster.CreateWithKubeconfigPath(flags.Kubeconfig),
	); err != nil {
		if errs := errors.Errors(err); errs != nil {
			for _, problem := range errs {
				logger.Errorf("%v", problem)
			}
			return errors.New("aborting due to invalid configuration")
		}
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}
