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
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	kindDefaults "sigs.k8s.io/kind/pkg/apis/config/defaults"
	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"
)

// VersionInfo contains structured version information
type VersionInfo struct {
	Kind KindInfo `json:"kind" yaml:"kind"`
}

// KindInfo contains kind CLI version details
type KindInfo struct {
	Version      string `json:"version" yaml:"version"`
	Platform     string `json:"platform" yaml:"platform"`
	DefaultImage string `json:"defaultImage" yaml:"defaultImage"`
}

// Version returns the kind CLI Semantic Version
func Version() string {
	return version(versionCore, versionPreRelease, gitCommit, gitCommitCount)
}

func version(core, preRelease, commit, commitCount string) string {
	v := core
	// add pre-release version info if we have it
	if preRelease != "" {
		v += "-" + preRelease
		// If commitCount was set, add to the pre-release version
		if commitCount != "" {
			v += "." + commitCount
		}
		// if commit was set, add the + <build>
		// we only do this for pre-release versions
		if commit != "" {
			// NOTE: use 14 character short hash, like Kubernetes
			v += "+" + truncate(commit, 14)
		}
	}
	return v
}

// DisplayVersion is Version() display formatted, this is what the version
// subcommand prints
func DisplayVersion() string {
	return "kind v" + Version() + " " + runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH
}

// versionCore is the core portion of the kind CLI version per Semantic Versioning 2.0.0
const versionCore = "0.32.0"

// versionPreRelease is the base pre-release portion of the kind CLI version per
// Semantic Versioning 2.0.0
var versionPreRelease = "alpha"

// gitCommitCount count the commits since the last release.
// It is injected at build time.
var gitCommitCount = ""

// gitCommit is the commit used to build the kind binary, if available.
// It is injected at build time.
var gitCommit = ""

// NewCommand returns a new cobra.Command for version
func NewCommand(logger log.Logger, streams cmd.IOStreams) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the kind CLI version",
		Long:  "Prints the kind CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger, streams, output)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "output format (json or yaml)")
	return cmd
}

func runE(logger log.Logger, streams cmd.IOStreams, output string) error {
	switch output {
	case "json":
		return printJSON(streams)
	case "yaml":
		return printYAML(streams)
	case "":
		if logger.V(0).Enabled() {
			// if not -q / --quiet, show lots of info
			fmt.Fprintln(streams.Out, DisplayVersion())
		} else {
			// otherwise only show semver
			fmt.Fprintln(streams.Out, Version())
		}
		return nil
	default:
		return fmt.Errorf("invalid output format: %q (must be json or yaml)", output)
	}
}

func getVersionInfo() VersionInfo {
	return VersionInfo{
		Kind: KindInfo{
			Version:      "v" + Version(),
			Platform:     runtime.GOOS + "/" + runtime.GOARCH,
			DefaultImage: kindDefaults.Image,
		},
	}
}

func printJSON(streams cmd.IOStreams) error {
	info := getVersionInfo()
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func printYAML(streams cmd.IOStreams) error {
	info := getVersionInfo()
	data, err := yaml.Marshal(info)
	if err != nil {
		return err
	}
	fmt.Fprint(streams.Out, string(data))
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) < maxLen {
		return s
	}
	return s[:maxLen]
}
