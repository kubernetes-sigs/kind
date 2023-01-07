/*
Copyright 2023 The Kubernetes Authors.

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

// Package provider implements the `provider` command
package provider

import (
	"fmt"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/internal/runtime"
	"sigs.k8s.io/kind/pkg/log"
)

// NewCommand returns a new cobra.Command for provider
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	var current bool
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "provider",
		Short: "Display information around providers",
		Long:  "Display information on supported providers, and currently active provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, streams, current)
		},
	}
	cmd.Flags().BoolVarP(
		&current,
		"current",
		"c",
		false,
		"display the current active provider",
	)
	return cmd
}

func runE(_ log.Logger, streams cmd.IOStreams, current bool) error {
	if current {
		provider := cluster.NewProvider(
			runtime.GetDefault(log.NoopLogger{}),
		)
		fmt.Fprintln(streams.Out, provider.Name())
		return nil
	}

	providers := cluster.SupportedProviders()
	for _, provider := range providers {
		fmt.Fprintln(streams.Out, provider.Name())
	}
	return nil
}
