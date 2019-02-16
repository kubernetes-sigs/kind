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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/fs"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the cluster logs
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: cobra.MaximumNArgs(1),
		// TODO(bentheelder): more detailed usage
		Use:   "logs [output-dir]",
		Short: "exports logs to to a tempdir or [output-dir] if specified",
		Long:  "exports logs to to a tempdir or [output-dir] if specified",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "the cluster context name")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// Check if the cluster name exists
	known, err := cluster.IsKnown(flags.Name)
	if err != nil {
		return err
	}
	if !known {
		return errors.Errorf("unknown cluster %q", flags.Name)
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
	context := cluster.NewContext(flags.Name)
	if err := context.CollectLogs(dir); err != nil {
		return err
	}
	fmt.Println("Exported logs to: " + dir)
	return nil
}
