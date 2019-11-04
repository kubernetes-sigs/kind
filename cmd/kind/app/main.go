package app

import (
	"os"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/cmd/kind"
	"sigs.k8s.io/kind/pkg/log"
)

// Main is the kind main(), it will invoke Run(), if an error is returned
// it will then call os.Exit
func Main() {
	if err := Run(cmd.NewLogger(), cmd.StandardIOStreams()); err != nil {
		os.Exit(1)
	}
}

// Run invokes the kind root command, returning the error.
// See: sigs.k8s.io/kind/pkg/cmd/kind
func Run(logger log.Logger, streams cmd.IOStreams) error {
	return kind.NewCommand(logger, streams).Execute()
}
