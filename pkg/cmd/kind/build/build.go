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

// Package build implements the `build` command
package build

import (
	"errors"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/cmd/kind/build/nodeimage"
	"sigs.k8s.io/kind/pkg/log"
)

// NewCommand returns a new cobra.Command for building
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "build",
		Short: "Build one of [node-image]",
		Long:  "Build one of [node-image]",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.Help()
			if err != nil {
				return err
			}
			return errors.New("Subcommand is required")
		},
	}
	// add subcommands
	cmd.AddCommand(nodeimage.NewCommand(logger, streams))
	return cmd
}
