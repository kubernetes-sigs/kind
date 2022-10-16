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

package version

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Value     string
		MaxLength int
		Expected  string
	}{
		{
			Value:     "A Really Long String",
			MaxLength: 1,
			Expected:  "A",
		},
		{
			Value:     "A Short String",
			MaxLength: 10,
			Expected:  "A Short St",
		},
		{
			Value:     "Under Max Length String",
			MaxLength: 1000,
			Expected:  "Under Max Length String",
		},
	}
	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.Value, func(t *testing.T) {
			t.Parallel()
			result := truncate(tc.Value, tc.MaxLength)
			// sanity check length
			if len(result) > tc.MaxLength {
				t.Errorf("Result %q longer than Max Length %d!", result, tc.MaxLength)
			}
			if tc.Expected != result {
				t.Errorf("Strings did not match!")
				t.Errorf("Expected: %q", tc.Expected)
				t.Errorf("But got: %q", result)
			}
		})
	}
}

func TestVersion_SuccessWithGitCommitCountAndHash(t *testing.T) {

	if versionPreRelease != "" {
		gitCommitBackup := gitCommit
		gitCommitCountBackup := gitCommitCount
		defer func() {
			gitCommit = gitCommitBackup
			gitCommitCount = gitCommitCountBackup
		}()
		gitCommit = "mocked-hash"
		gitCommitCount = "mocked-count"
		want := versionCore + "-" + versionPreRelease + "." + gitCommitCount + "+" + gitCommit

		got := Version()

		if got != want {
			t.Errorf("Version() = %v, want %v", got, want)
		}
	}
}

func TestVersion_SuccessWithGitCommitCountAndNoHash(t *testing.T) {

	if versionPreRelease != "" {
		gitCommitBackup := gitCommit
		gitCommitCountBackup := gitCommitCount
		defer func() {
			gitCommit = gitCommitBackup
			gitCommitCount = gitCommitCountBackup
		}()
		gitCommit = ""
		gitCommitCount = "mocked-count"
		want := versionCore + "-" + versionPreRelease + "." + gitCommitCount

		got := Version()

		if got != want {
			t.Errorf("Version() = %v, want %v", got, want)
		}
	}
}

func TestVersion_SuccessWithGitHashAndNoCount(t *testing.T) {

	if versionPreRelease != "" {
		gitCommitBackup := gitCommit
		gitCommitCountBackup := gitCommitCount
		defer func() {
			gitCommit = gitCommitBackup
			gitCommitCount = gitCommitCountBackup
		}()
		gitCommit = "mocked-hash"
		gitCommitCount = ""
		want := versionCore + "-" + versionPreRelease + "+" + gitCommit

		got := Version()

		if got != want {
			t.Errorf("Version() = %v, want %v", got, want)
		}
	}
}
