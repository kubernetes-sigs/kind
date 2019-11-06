/*
Copyright 2019 The Kubernetes Authors.

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

package app

import (
	"testing"
)

func TestCheckQuiet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name        string
		Args        []string
		ExpectQuiet bool
	}{
		// normal cases, we expect it to be set
		{
			Name:        "simply q",
			Args:        []string{"-q"},
			ExpectQuiet: true,
		},
		{
			Name:        "simply quiet",
			Args:        []string{"--quiet"},
			ExpectQuiet: true,
		},
		{
			Name:        "all quiet on the cli",
			Args:        []string{"all", "quiet", "on", "the", "cli", "--quiet"},
			ExpectQuiet: true,
		},
		// pflag will throw an ErrHelp when -h / --help are in args even though
		// we don't register these as flags, checkQuiet should ignore them
		{
			Name:        "with ignored help",
			Args:        []string{"--quiet", "--help"},
			ExpectQuiet: true,
		},
		{
			Name:        "with ignored h",
			Args:        []string{"--quiet", "-h"},
			ExpectQuiet: true,
		},
		// not quiet for these cases ...
		{
			Name:        "no args",
			Args:        []string{},
			ExpectQuiet: false,
		},
		{
			Name:        "loud",
			Args:        []string{"--loud"},
			ExpectQuiet: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result := checkQuiet(tc.Args)
			if result != tc.ExpectQuiet {
				t.Fatalf("fooo")
			}
		})
	}
}
