/*
Copyright 2020 The FlowQ Authors.

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

// Package server implements the `server` command
package server

import (
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/log"
	"sigs.k8s.io/kind/pkg/server"
)

type flagpole struct {
	Address string
	Port    string
}

// NewCommand returns a new cobra.Command for version
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "server",
		Short: "run kind as REST API Server",
		Long:  "run kind as REST API Server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.OverrideDefaultName(cmd.Flags())
			return runE(logger, streams, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Address, "Host", "127.0.0.1", "server listen port, config (default 127.0.0.1)")
	cmd.Flags().StringVar(&flags.Port, "port", "8000", "server listen port, config (default 8000)")

	return cmd
}

func runE(logger log.Logger, streams cmd.IOStreams, flags *flagpole) error {
	server.APIServerStart(logger, flags.Address, flags.Port)
	return nil
}
