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

// Package env implements the `env` command.
package env

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"
)

type environmentVariable struct {
	Name        string
	Description string
}

var environmentVariables = []environmentVariable{
	{
		Name:        "KIND_CLUSTER_NAME",
		Description: "Default cluster name for create/delete commands",
	},
	{
		Name:        "KIND_EXPERIMENTAL_PROVIDER",
		Description: "Selects the runtime provider used for new clusters",
	},
	{
		Name:        "KIND_EXPERIMENTAL_DOCKER_NETWORK",
		Description: "Overrides the Docker network used for nodes",
	},
	{
		Name:        "KIND_EXPERIMENTAL_PODMAN_NETWORK",
		Description: "Overrides the Podman network used for nodes",
	},
	{
		Name:        "KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER",
		Description: "Passes a custom containerd snapshotter to nodes",
	},
	{
		Name:        "KIND_DNS_SEARCH",
		Description: "Provides DNS search domains for nodes",
	},
	{
		Name:        "KUBECONFIG",
		Description: "Overrides the kubeconfig path used by kind",
	},
}

// NewCommand returns a new cobra.Command for env.
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "Prints the kind environment variables",
		Long:  "Prints the kind environment variables and their current values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printEnv(streams.Out)
		},
	}
}

func printEnv(out io.Writer) error {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, err := fmt.Fprintln(tw, "NAME\tVALUE\tDESCRIPTION")
	if err != nil {
		return err
	}
	for _, env := range environmentVariables {
		value, ok := os.LookupEnv(env.Name)
		if !ok {
			value = "<unset>"
		} else if value == "" {
			value = "<empty>"
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", env.Name, value, env.Description); err != nil {
			return err
		}
	}
	return tw.Flush()
}
