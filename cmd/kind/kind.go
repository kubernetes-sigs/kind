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

// Package kind implements the root kind cobra command, and the cli Main()
package kind

import (
	goflag "flag"
	"os"

	//flag "github.com/spf13/pflag"
	"github.com/spf13/cobra"
	log "k8s.io/klog"

	"sigs.k8s.io/kind/cmd/kind/build"
	"sigs.k8s.io/kind/cmd/kind/create"
	"sigs.k8s.io/kind/cmd/kind/delete"
	"sigs.k8s.io/kind/cmd/kind/get"
)

// NewCommand returns a new cobra.Command implementing the root command for kind
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kind",
		Short: "kind is a tool for managing local Kubernetes clusters",
		Long:  "kind creates and manages local Kubernetes clusters using Docker container 'nodes'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// add all top level subcommands
	cmd.AddCommand(build.NewCommand())
	cmd.AddCommand(create.NewCommand())
	cmd.AddCommand(delete.NewCommand())
	cmd.AddCommand(get.NewCommand())
	return cmd
}

// Main creates the root command for kind, sets up klog, and serves as
// entrypoint.
func Main() {
	cmd := NewCommand()
	// setup klog.
	klogFlags := goflag.NewFlagSet("kind", goflag.ContinueOnError)
	log.InitFlags(klogFlags)
	cmd.PersistentFlags().AddGoFlagSet(klogFlags)

	// now we can run.
	if err := cmd.Execute(); err != nil {
		log.Flush()
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(-1)
	}
	log.Flush()
}
