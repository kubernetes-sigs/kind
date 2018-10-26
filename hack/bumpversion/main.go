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

// simple command to bump kind's semver
// consumes VERSION or the current git tag
// consume PRERELEASE or the prerelease info in the current version

package main

import (
	"fmt"
	"os"

	"github.com/coreos/go-semver/semver"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kind/pkg/exec"
)

func getVersion() (string, error) {
	// first check for the env
	version := os.Getenv("VERSION")
	if version != "" {
		return version, nil
	}
	// then query git instead
	cmd := exec.Command("git", "describe", "--tags")
	lines, err := cmd.CombinedOutputLines()
	if err != nil {
		return "", err
	}
	return lines[0], nil
}

func failIfErr(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(-1)
	}
}

func bumpVersion(version, bumpType string, prerelease string) (string, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	// bump
	switch bumpType {
	case "major":
		v.BumpPatch()
	case "minor":
		v.BumpPatch()
	case "patch":
		v.BumpPatch()
	case "":
		break
	default:
		return "", fmt.Errorf("unkown bump type: %s", bumpType)
	}
	// NOTE: .Bump*() does this, but in the no-op bump case we also
	// want to do this and set prerelease ourselves
	// our version do not use metadata currently
	// TODO(bentheelder): fix this if we ever use metadata etc..
	v.PreRelease = semver.PreRelease("")
	v.Metadata = ""
	// add pre-release if specified
	if prerelease != "" {
		v.PreRelease = semver.PreRelease("alpha")
	}
	return v.String(), nil
}

func main() {
	bumpType := pflag.String("type", "patch", "version bump type")
	prerelease := pflag.String("prerelease", "alpha", "the prerelease field")
	pflag.Parse()

	version, err := getVersion()
	failIfErr(err)

	bumped, err := bumpVersion(version, *bumpType, *prerelease)
	failIfErr(err)

	println(bumped)
}
