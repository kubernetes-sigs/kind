/*
Copyright 2019 The Kubernetes Authors.

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

package app

import (
	"io"
	"os"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/cmd/kind"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// Main is the kind main(), it will invoke Run(), if an error is returned
// it will then call os.Exit
func Main() {
	if err := Run(cmd.NewLogger(), cmd.StandardIOStreams(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

// Run invokes the kind root command, returning the error.
// See: sigs.k8s.io/kind/pkg/cmd/kind
func Run(logger log.Logger, streams cmd.IOStreams, args []string) error {
	// NOTE: we handle the quiet flag here so we can fully silence cobra
	if checkQuiet(args) {
		// if we are in quiet mode, we want to suppress all status output
		// only streams.Out should be written to (program output)
		logger = log.NoopLogger{}
		streams.ErrOut = io.Discard
	}
	// actually run the command
	c := kind.NewCommand(logger, streams)
	c.SetArgs(args)
	if err := c.Execute(); err != nil {
		logError(logger, err)
		return err
	}
	return nil
}

// checkQuiet returns true if -q / --quiet was set in args
func checkQuiet(args []string) bool {
	flags := pflag.NewFlagSet("persistent-quiet", pflag.ContinueOnError)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	quiet := false
	flags.BoolVarP(
		&quiet,
		"quiet",
		"q",
		false,
		"silence all stderr output",
	)
	// NOTE: pflag will error if -h / --help is specified
	// We don't care here. That will be handled downstream
	// It will also call flags.Usage so we're making that no-op
	flags.Usage = func() {}
	_ = flags.Parse(args)
	return quiet
}

// logError logs the error and the root stacktrace if there is one
func logError(logger log.Logger, err error) {
	colorEnabled := cmd.ColorEnabled(logger)
	if colorEnabled {
		logger.Errorf("\x1b[31mERROR\x1b[0m: %v", err)
	} else {
		logger.Errorf("ERROR: %v", err)
	}
	// Display Output if the error was from running a command ...
	if err := exec.RunErrorForError(err); err != nil {
		if colorEnabled {
			logger.Errorf("\x1b[31mCommand Output\x1b[0m: %s", err.Output)
		} else {
			logger.Errorf("\nCommand Output: %s", err.Output)
		}
	}
	// TODO: stacktrace should probably be guarded by a higher level ...?
	if logger.V(1).Enabled() {
		// Then display stack trace if any (there should be one...)
		if trace := errors.StackTrace(err); trace != nil {
			if colorEnabled {
				logger.Errorf("\x1b[31mStack Trace\x1b[0m: %+v", trace)
			} else {
				logger.Errorf("\nStack Trace: %+v", trace)
			}
		}
	}
}
