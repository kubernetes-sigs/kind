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
	"github.com/spf13/cobra"

	"k8s.io/test-infra/kind/cmd/kind/build/base"
	"k8s.io/test-infra/kind/cmd/kind/build/node"
)

// NewCommand returns a new cobra.Command for building
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		// TODO(bentheelder): more detailed usage
		Use:   "build",
		Short: "build",
		Long:  "build",
	}
	// add subcommands
	cmd.AddCommand(base.NewCommand())
	cmd.AddCommand(node.NewCommand())
	return cmd
}
