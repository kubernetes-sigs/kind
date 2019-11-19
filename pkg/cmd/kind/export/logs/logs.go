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

// Package logs implements the `logs` command
package logs

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/fs"
	"sigs.k8s.io/kind/pkg/internal/util/cli"
	"sigs.k8s.io/kind/pkg/log"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the cluster logs
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: cobra.MaximumNArgs(1),
		// TODO(bentheelder): more detailed usage
		Use:   "logs [output-dir]",
		Short: "exports logs to a tempdir or [output-dir] if specified",
		Long:  "exports logs to a tempdir or [output-dir] if specified",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, logger, streams, flags, args)
		},
	}
	cmd.Flags().String("name", "", fmt.Sprintf(`the cluster name. Overrides KIND_CLUSTER_NAME environment variable (default "%s")`, cluster.DefaultName))
	return cmd
}

func runE(cmd *cobra.Command, logger log.Logger, streams cmd.IOStreams, flags *flagpole, args []string) error {
	provider := cluster.NewProvider(
		cluster.ProviderWithLogger(logger),
	)

	flags.Name = cli.GetClusterNameFlags(cmd)
	// Check if the cluster has any running nodes
	nodes, err := provider.ListNodes(flags.Name)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return fmt.Errorf("unknown cluster %q", flags.Name)
	}

	// get the optional directory argument, or create a tempdir
	var dir string
	if len(args) == 0 {
		t, err := fs.TempDir("", "")
		if err != nil {
			return err
		}
		dir = t
	} else {
		dir = args[0]
	}

	// collect the logs
	if err := provider.CollectLogs(flags.Name, dir); err != nil {
		return err
	}

	logger.V(0).Infof("Exported logs for cluster %q to:", flags.Name)
	fmt.Fprintln(streams.Out, dir)
	return nil
}
