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

// Package kubeconfigpath implements the `kubeconfig-path` command
package kubeconfigpath

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/cluster/kubeconfig"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the kubeconfig-path
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kubeconfig-path",
		Short: "DEPRECATED: prints the default kubeconfig path for the kind cluster by --name",
		Long:  `DEPRECATED: prints the default kubeconfig path for the kind cluster by --name`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, streams)
		},
	}
	// NOTE: we need to support this flag but it's no longer used in the path
	cmd.Flags().StringVar(
		&flags.Name,
		"name",
		cluster.DefaultName,
		"the cluster context name",
	)
	return cmd
}

func runE(logger log.Logger, streams cmd.IOStreams) error {
	logger.Warn("`kind get kubeconfig-path` is deprecated!")
	logger.Warn("")
	logger.Warn("KIND will export and merge kubeconfig like kops, minikube, etc.")
	logger.Warn("This command is now unnecessary and will be removed in a future release.")
	logger.Warn("")
	logger.Warn("For more info see: https://github.com/kubernetes-sigs/kind/issues/1060")
	logger.Warn("See also the output of `kind create cluster`")
	logger.Warn("")
	fmt.Fprintln(streams.Out, kubeconfig.LegacyPath())
	return nil
}
