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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/cmd/kind/build"
	"sigs.k8s.io/kind/cmd/kind/completion"
	"sigs.k8s.io/kind/cmd/kind/create"
	"sigs.k8s.io/kind/cmd/kind/delete"
	"sigs.k8s.io/kind/cmd/kind/export"
	"sigs.k8s.io/kind/cmd/kind/get"
	"sigs.k8s.io/kind/cmd/kind/load"
	"sigs.k8s.io/kind/cmd/kind/version"
	"sigs.k8s.io/kind/pkg/globals"
	"sigs.k8s.io/kind/pkg/log"
)

// Flags for the kind command
type Flags struct {
	LogLevel  string
	Verbosity int32
	Quiet     bool
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
			return runE(flags, cmd)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}
	cmd.PersistentFlags().StringVar(
		&flags.LogLevel,
		"loglevel",
		"",
		"DEPRECATED: see -v instead",
	)
	cmd.PersistentFlags().Int32VarP(
		&flags.Verbosity,
		"verbosity",
		"v",
		0,
		"info log verbosity",
	)
	cmd.PersistentFlags().BoolVarP(
		&flags.Quiet,
		"quiet",
		"q",
		false,
		"silence all stderr output",
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

func runE(flags *Flags, cmd *cobra.Command) error {
	// handle limited migration for --loglevel
	setLogLevel := cmd.Flag("loglevel").Changed
	setVerbosity := cmd.Flag("verbosity").Changed
	if setLogLevel && !setVerbosity {
		switch flags.LogLevel {
		case "debug":
			flags.Verbosity = 3
		case "trace":
			flags.Verbosity = 2147483647
		}
	}
	// normal logger setup
	if flags.Quiet {
		globals.SetLogger(log.NoopLogger{})
	} else {
		globals.UseCLILogger(os.Stderr, log.Level(flags.Verbosity))
	}
	// warn about deprecated flag if used
	if setLogLevel {
		globals.GetLogger().Warn("WARNING: --loglevel is deprecated, please switch to -v and -q!")
	}
	return nil
}

// Run runs the `kind` root command
func Run() error {
	return NewCommand().Execute()
}

// Main wraps Run and sets the log formatter
func Main() {
	if err := Run(); err != nil {
		logError(err)
		os.Exit(1)
	}
}

// logError logs the error and the root stacktrace if there is one
func logError(err error) {
	globals.GetLogger().Errorf("ERROR: %v", err)
	// if debugging is enabled (non-zero verbosity), display stack trace if any
	if globals.GetLogger().V(1).Enabled() {
		if trace := stackTrace(err); trace != nil {
			globals.GetLogger().Errorf("%+v", trace)
		}
	}
}

// stackTrace returns the deepest StackTrace is a Cause chain
// https://github.com/pkg/errors/issues/173
func stackTrace(err error) errors.StackTrace {
	// github.com/pkg/errors errors type interfaces
	type causer interface {
		Cause() error
	}
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	// walk all causes, keeping the last one with a StackTrace
	var stackErr error
	for {
		if _, ok := err.(stackTracer); ok {
			stackErr = err
		}
		if causerErr, ok := err.(causer); ok {
			err = causerErr.Cause()
		} else {
			break
		}
	}

	if stackErr != nil {
		return stackErr.(stackTracer).StackTrace()
	}
	return nil
}
