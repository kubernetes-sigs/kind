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

// Package export implements the `export` command
package export

import (
	"github.com/spf13/cobra"

	"sigs.k8s.io/kind/cmd/kind/export/logs"
)

// NewCommand returns a new cobra.Command for export
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "export",
		Short: "exports one of [logs]",
		Long:  "exports one of [logs]",
	}
	// add subcommands
	cmd.AddCommand(logs.NewCommand())
	return cmd
}
