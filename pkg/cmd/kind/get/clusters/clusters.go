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

// Package clusters implements the `clusters` command
package clusters

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/runtime"
)

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "clusters",
		Short: "Lists existing kind clusters by their name",
		Long:  "Lists existing kind clusters by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, streams)
		},
	}
	return cmd
}

func runE(logger log.Logger, streams cmd.IOStreams) error {
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
		runtime.GetDefault(logger),
	)
	clusters, err := provider.List()
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		logger.V(0).Info("No kind clusters found.")
		return nil
	}
	for _, cluster := range clusters {
		fmt.Fprintln(streams.Out, cluster)
	}
	return nil
}
