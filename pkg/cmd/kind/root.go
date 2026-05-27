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
	"errors"
	"io"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/cmd/kind/build"
	"sigs.k8s.io/kind/pkg/cmd/kind/completion"
	"sigs.k8s.io/kind/pkg/cmd/kind/create"
	"sigs.k8s.io/kind/pkg/cmd/kind/delete"
	"sigs.k8s.io/kind/pkg/cmd/kind/export"
	"sigs.k8s.io/kind/pkg/cmd/kind/get"
	"sigs.k8s.io/kind/pkg/cmd/kind/load"
	"sigs.k8s.io/kind/pkg/cmd/kind/version"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Verbosity      int32
	Quiet          bool
	Multihost      bool
	Hosts          string
	BootstrapSwarm bool
}

// NewCommand returns a new cobra.Command implementing the root command for kind
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "kind",
		Short: "kind is a tool for managing local Kubernetes clusters",
		Long:  "kind creates and manages local Kubernetes clusters using Docker container 'nodes'",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, flags)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version(),
	}
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)
	cmd.PersistentFlags().Int32VarP(
		&flags.Verbosity,
		"verbosity",
		"v",
		0,
		"info log verbosity, higher value produces more output",
	)
	cmd.PersistentFlags().BoolVarP(
		&flags.Quiet,
		"quiet",
		"q",
		false,
		"silence all stderr output",
	)
	cmd.PersistentFlags().BoolVar(
		&flags.Multihost,
		"multihost",
		false,
		"enable experimental multi-host support (distribute nodes across multiple Docker daemons via Swarm)",
	)
	cmd.PersistentFlags().StringVar(
		&flags.Hosts,
		"hosts",
		"",
		"multihost: comma-separated <docker-context>=<external-addr> pairs (first = manager)",
	)
	cmd.PersistentFlags().BoolVar(
		&flags.BootstrapSwarm,
		"bootstrap-swarm",
		false,
		"multihost: run 'docker swarm init' on the manager and 'swarm join' on each worker before creating the cluster",
	)
	// add all top level subcommands
	cmd.AddCommand(build.NewCommand(logger, streams))
	cmd.AddCommand(completion.NewCommand(logger, streams))
	cmd.AddCommand(create.NewCommand(logger, streams))
	cmd.AddCommand(delete.NewCommand(logger, streams))
	cmd.AddCommand(export.NewCommand(logger, streams))
	cmd.AddCommand(get.NewCommand(logger, streams))
	cmd.AddCommand(version.NewCommand(logger, streams))
	cmd.AddCommand(load.NewCommand(logger, streams))
	return cmd
}

func runE(logger log.Logger, flags *flagpole) error {
	// normal logger setup
	if flags.Quiet {
		// NOTE: if we are coming from app.Run handling this flag is
		// redundant, however it doesn't hurt, and this may be called directly.
		maybeSetWriter(logger, io.Discard)
	}
	maybeSetVerbosity(logger, log.Level(flags.Verbosity))
	if flags.Multihost {
		if err := handleMultihost(flags); err != nil {
			return err
		}
	}
	return nil
}

// handleMultihost wires the --multihost/--hosts/--bootstrap-swarm flags into
// the runtime by setting the equivalent KIND_EXPERIMENTAL_PROVIDER / KIND_HOSTS
// environment variables that pkg/internal/runtime picks up later.  Done this
// way so we don't need to thread the flag through every subcommand.
func handleMultihost(flags *flagpole) error {
	if flags.Hosts == "" && os.Getenv("KIND_HOSTS") == "" {
		return errors.New("--multihost requires --hosts <ctx>=<addr>[,...] or the KIND_HOSTS env var")
	}
	if flags.Hosts != "" {
		os.Setenv("KIND_HOSTS", flags.Hosts)
	}
	os.Setenv("KIND_EXPERIMENTAL_PROVIDER", "swarm")
	if flags.BootstrapSwarm {
		os.Setenv("KIND_BOOTSTRAP_SWARM", "1")
	}
	return nil
}

// maybeSetWriter will call logger.SetWriter(w) if logger has a SetWriter method
func maybeSetWriter(logger log.Logger, w io.Writer) {
	type writerSetter interface {
		SetWriter(io.Writer)
	}
	v, ok := logger.(writerSetter)
	if ok {
		v.SetWriter(w)
	}
}

// maybeSetVerbosity will call logger.SetVerbosity(verbosity) if logger
// has a SetVerbosity method
func maybeSetVerbosity(logger log.Logger, verbosity log.Level) {
	type verboser interface {
		SetVerbosity(log.Level)
	}
	v, ok := logger.(verboser)
	if ok {
		v.SetVerbosity(verbosity)
	}
}
