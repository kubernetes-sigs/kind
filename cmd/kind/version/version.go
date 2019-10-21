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

// Version returns the kind CLI Semantic Version
func Version() string {
	v := VersionCore
	// add pre-release version info if we have it
	if VersionPreRelease != "" {
		v += "-" + VersionPreRelease
		// if commit was set, add the + <build>
		// we only do this for pre-release versions
		if GitCommit != "" {
			v += "+" + GitCommit
		}
	}
	return v
}

// VersionCore is the core portion of the kind CLI version per Semantic Versioning 2.0.0
const VersionCore = "v0.6.0"

// VersionPreRelease is the pre-release portion of the kind CLI version per
// Semantic Versioning 2.0.0
const VersionPreRelease = "alpha"

// GitCommit is the commit used to build the kind binary, if available.
// It is injected at build time.
var GitCommit = ""

// NewCommand returns a new cobra.Command for version
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "version",
		Short: "prints the kind CLI version",
		Long:  "prints the kind CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Version())
			return nil
		},
	}
	return cmd
}
