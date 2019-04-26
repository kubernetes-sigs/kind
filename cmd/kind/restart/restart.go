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

// Package restart implements the `restart` command
package restart

import (
	"github.com/spf13/cobra"

	restartcluster "sigs.k8s.io/kind/cmd/kind/restart/cluster"
)

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "restart",
		Short: "restarts one of [cluster]",
		Long:  "restarts one of local Kubernetes cluster (cluster)",
	}
	cmd.AddCommand(restartcluster.NewCommand())
	return cmd
}
