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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8s.io/test-infra/kind/cmd/kind/build"
	"k8s.io/test-infra/kind/cmd/kind/create"
	"k8s.io/test-infra/kind/cmd/kind/delete"
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
	return NewCommand().Execute()
}

// Main wraps Run, adding a log.Fatal(err) on error, and setting the log formatter
func Main() {
	// this formatter is the default, but the timestamps output aren't
	// particularly useful, they're relative to the command start
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05-0700",
	})
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
