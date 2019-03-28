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
	"os"

	"github.com/sirupsen/logrus"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/cmd/kind/build"
	"sigs.k8s.io/kind/cmd/kind/completion"
	"sigs.k8s.io/kind/cmd/kind/create"
	"sigs.k8s.io/kind/cmd/kind/delete"
	"sigs.k8s.io/kind/cmd/kind/export"
	"sigs.k8s.io/kind/cmd/kind/get"
	"sigs.k8s.io/kind/cmd/kind/load"
	"sigs.k8s.io/kind/cmd/kind/version"
	logutil "sigs.k8s.io/kind/pkg/log"
)

const defaultLevel = logrus.WarnLevel

// Flags for the kind command
type Flags struct {
	LogLevel string
}

// NewCommand returns a new cobra.Command implementing the root command for kind
func NewCommand() *cobra.Command {
	flags := &Flags{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kind",
		Short: "kind is a tool for managing local Kubernetes clusters",
		Long:  "kind creates and manages local Kubernetes clusters using Docker container 'nodes'",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
		SilenceUsage: true,
		Version:      version.Version,
	}
	cmd.PersistentFlags().StringVar(
		&flags.LogLevel,
		"loglevel",
		defaultLevel.String(),
		"logrus log level "+logutil.LevelsString(),
	)
	// add all top level subcommands
	cmd.AddCommand(build.NewCommand())
	cmd.AddCommand(completion.NewCommand())
	cmd.AddCommand(create.NewCommand())
	cmd.AddCommand(delete.NewCommand())
	cmd.AddCommand(export.NewCommand())
	cmd.AddCommand(get.NewCommand())
	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(load.NewCommand())
	return cmd
}

func runE(flags *Flags, cmd *cobra.Command, args []string) error {
	level := defaultLevel
	parsed, err := log.ParseLevel(flags.LogLevel)
	if err != nil {
		log.Warnf("Invalid log level '%s', defaulting to '%s'", flags.LogLevel, level)
	} else {
		level = parsed
	}
	log.SetLevel(level)
	return nil
}

// Run runs the `kind` root command
func Run() error {
	return NewCommand().Execute()
}

// Main wraps Run and sets the log formatter
func Main() {
	// let's explicitly set stdout
	log.SetOutput(os.Stdout)
	// this formatter is the default, but the timestamps output aren't
	// particularly useful, they're relative to the command start
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
		// we force colors because this only forces over the isTerminal check
		// and this will not be accurately checkable later on when we wrap
		// the logger output with our logutil.StatusFriendlyWriter
		ForceColors: logutil.IsTerminal(log.StandardLogger().Out),
	})
	if err := Run(); err != nil {
		os.Exit(1)
	}
}
