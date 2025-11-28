/*
Copyright 2024 The Kubernetes Authors.

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

// Package env implements the `env` command.
package env

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"
)

type envVar struct {
	Name        string
	Description string
}

var envVariables = []envVar{
	{
		Name:        "KIND_EXPERIMENTAL_PROVIDER",
		Description: "Override provider auto-detection (docker, podman, nerdctl).",
	},
	{
		Name:        "KIND_CLUSTER_NAME",
		Description: "Default cluster name used by commands (default \"kind\").",
	},
	{
		Name:        "KUBECONFIG",
		Description: "Path to the kubeconfig file(s) for cluster access.",
	},
	{
		Name:        "KIND_EXPERIMENTAL_DOCKER_NETWORK",
		Description: "Docker network to attach cluster nodes to.",
	},
	{
		Name:        "KIND_EXPERIMENTAL_PODMAN_NETWORK",
		Description: "Podman network to attach cluster nodes to.",
	},
	{
		Name:        "KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER",
		Description: "Containerd snapshotter inside nodes (e.g. overlayfs, fuse-overlayfs).",
	},
}

// NewCommand returns a new cobra.Command for env.
func NewCommand(_ log.Logger, streams cmd.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Prints key environment variables used by kind",
		Long:  "Prints key environment variables used by kind along with their current values.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, variable := range envVariables {
				value := os.Getenv(variable.Name)
				if _, err := fmt.Fprintf(streams.Out, "%s=%q\t%s\n", variable.Name, value, variable.Description); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
