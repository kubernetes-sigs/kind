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

// Package kubeconfig implements the `kubeconfig` command
package kubeconfig

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/internal/util/cli"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Name       string
	Kubeconfig string
}

// NewCommand returns a new cobra.Command for exporting the kubeconfig
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kubeconfig",
		Short: "exports cluster kubeconfig",
		Long:  "exports cluster kubeconfig",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, logger, flags)
		},
	}
	cmd.Flags().String(
		"name",
		"",
		fmt.Sprintf(`the cluster name. Overrides KIND_CLUSTER_NAME environment variable (default "%s")`, cluster.DefaultName),
	)
	cmd.Flags().StringVar(
		&flags.Kubeconfig,
		"kubeconfig",
		"",
		"sets kubeconfig path instead of $KUBECONFIG or $HOME/.kube/config",
	)
	return cmd
}

func runE(cmd *cobra.Command, logger log.Logger, flags *flagpole) error {
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)

	flags.Name = cli.GetClusterNameFlags(cmd)
	if err := provider.ExportKubeConfig(flags.Name, flags.Kubeconfig); err != nil {
		return err
	}
	// TODO: get kind-name from a method? OTOH we probably want to keep this
	// naming scheme stable anyhow...
	logger.V(0).Infof(`Set kubectl context to "kind-%s"`, flags.Name)
	return nil
}
