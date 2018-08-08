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

// Package cmd implements the kidna cobra command CLI
package cmd

import (
	"flag"

	"github.com/spf13/cobra"

	"k8s.io/test-infra/kind/cmd/kind/cmd/build"
	"k8s.io/test-infra/kind/cmd/kind/cmd/create"
	"k8s.io/test-infra/kind/cmd/kind/cmd/delete"
)

// NewCommand returns a new cobra.Command implementing the root command for kind
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kind",
		Short: "kind is a tool for managing local multi-node Kubernetes clusters",
		Long:  "kind creates and manages local multi-node Kubernetes clusters using Docker containers",
	}
	// add all top level subcommands
	cmd.AddCommand(build.NewCommand())
	cmd.AddCommand(create.NewCommand())
	cmd.AddCommand(delete.NewCommand())
	return cmd
}

// Run runs the `kind` root command
func Run() error {
	// Trick to avoid glog's 'logging before flag.Parse' warning
	flag.CommandLine.Parse([]string{})
	// glog logs to files by default, grr
	flag.Set("logtostderr", "true")

	cmd := NewCommand()
	// glog registers global flags on flag.CommandLine...
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	// actually execute the cobra commands now...
	return cmd.Execute()
}
