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

// Package version implements the `version` command
package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the kind CLI version
const Version = "v0.5.1"

// NewCommand returns a new cobra.Command for version
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "version",
		Short: "prints the kind CLI version",
		Long:  "prints the kind CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Version)
			return nil
		},
	}
	return cmd
}
