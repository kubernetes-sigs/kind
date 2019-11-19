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
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/internal/util/cli"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Name       string
	Kubeconfig string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "cluster",
		Short: "Deletes a cluster",
		Long:  "Deletes a resource",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, logger, flags)
		},
	}
	cmd.Flags().String("name", "", fmt.Sprintf(`the cluster name. Overrides KIND_CLUSTER_NAME environment variable (default "%s")`, cluster.DefaultName))
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", "", "sets kubeconfig path instead of $KUBECONFIG or $HOME/.kube/config")
	return cmd
}

func runE(cmd *cobra.Command, logger log.Logger, flags *flagpole) error {
	// Delete the cluster
	flags.Name = cli.GetClusterNameFlags(cmd)
	logger.V(0).Infof("Deleting cluster %q ...\n", flags.Name)
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)
	if err := provider.Delete(flags.Name, flags.Kubeconfig); err != nil {
		return errors.Wrap(err, "failed to delete cluster")
	}
	return nil
}
