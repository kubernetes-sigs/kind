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
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/internal/runtime"
)

type flagpole struct {
	Name       string
	Kubeconfig string
}

// NewCommand returns a new cobra.Command for cluster deletion
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Deletes a cluster",
		Long: `Deletes a Kind cluster from the system.

This is an idempotent operation, meaning it may be called multiple times without
failing (like "rm -f"). If the cluster resources exist they will be deleted, and
if the cluster is already gone it will just return success.

Errors will only occur if the cluster resources exist and are not able to be deleted.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.OverrideDefaultName(cmd.Flags())
			return deleteCluster(logger, flags)
		},
	}
	cmd.Flags().StringVarP(
		&flags.Name,
		"name",
		"n",
		cluster.DefaultName,
		"the cluster name",
	)
	cmd.Flags().StringVar(
		&flags.Kubeconfig,
		"kubeconfig",
		"",
		"sets kubeconfig path instead of $KUBECONFIG or $HOME/.kube/config",
	)
	return cmd
}

func deleteCluster(logger log.Logger, flags *flagpole) error {
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
		runtime.GetDefault(logger),
	)
	
	// Check if cluster exists before attempting deletion
	clusters, err := provider.List()
	if err != nil {
		return errors.Wrap(err, "failed to list clusters")
	}

	// Check if the target cluster exists
	clusterExists := false
	for _, clusterName := range clusters {
		if clusterName == flags.Name {
			clusterExists = true
			break
		}
	}

	// If using default name and no clusters exist, show helpful message
	if flags.Name == cluster.DefaultName && len(clusters) == 0 {
		logger.V(0).Infof("No kind clusters found.")
		return nil
	}

	// If cluster doesn't exist, show warning but don't fail (idempotent behavior)
	if !clusterExists {
		logger.V(0).Infof("Cluster %q not found. Available clusters: %v", flags.Name, clusters)
		return nil
	}

	// Delete individual cluster
	logger.V(0).Infof("Deleting cluster %q ...", flags.Name)
	if err := provider.Delete(flags.Name, flags.Kubeconfig); err != nil {
		return errors.Wrapf(err, "failed to delete cluster %q", flags.Name)
	}
	return nil
}
