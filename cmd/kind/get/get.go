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

// Package get implements the `get` command
package get

import (
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/cmd/kind/get/clusters"
	"sigs.k8s.io/kind/cmd/kind/get/kubeconfigpath"
)

// NewCommand returns a new cobra.Command for get
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "get",
		Short: "Gets one of [clusters, kubeconfig-path]",
		Long:  "Gets one of [clusters, kubeconfig-path]",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	// add subcommands
	cmd.AddCommand(clusters.NewCommand())
	cmd.AddCommand(kubeconfigpath.NewCommand())
	return cmd
}
