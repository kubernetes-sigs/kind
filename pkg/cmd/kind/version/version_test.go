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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"
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

// noopLogger implements log.Logger with no output, used in command tests
type noopLogger struct{}

func (n noopLogger) Warn(message string)                       {}
func (n noopLogger) Warnf(format string, args ...interface{})  {}
func (n noopLogger) Error(message string)                      {}
func (n noopLogger) Errorf(format string, args ...interface{}) {}
func (n noopLogger) V(level log.Level) log.InfoLogger          { return noopInfoLogger{} }

type noopInfoLogger struct{}

func (n noopInfoLogger) Info(message string)                      {}
func (n noopInfoLogger) Infof(format string, args ...interface{}) {}
func (n noopInfoLogger) Enabled() bool                            { return true }

func newTestStreams() (cmd.IOStreams, *bytes.Buffer) {
	var buf bytes.Buffer
	streams := cmd.IOStreams{Out: &buf}
	return streams, &buf
}

func TestVersionCommandOutputJSON(t *testing.T) {
	t.Parallel()
	streams, buf := newTestStreams()
	c := NewCommand(noopLogger{}, streams)
	c.SetArgs([]string{"-o", "json"})
	if err := c.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var info VersionInfo
	if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v\noutput: %s", err, buf.String())
	}
	if info.Version == "" {
		t.Error("expected non-empty Version field")
	}
	if info.Platform == "" {
		t.Error("expected non-empty Platform field")
	}
	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion field")
	}
	if info.DefaultImage == "" {
		t.Error("expected non-empty DefaultImage field")
	}
}

func TestVersionCommandOutputYAML(t *testing.T) {
	t.Parallel()
	streams, buf := newTestStreams()
	c := NewCommand(noopLogger{}, streams)
	c.SetArgs([]string{"-o", "yaml"})
	if err := c.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var info VersionInfo
	if err := yaml.Unmarshal(buf.Bytes(), &info); err != nil {
		t.Fatalf("failed to unmarshal YAML output: %v\noutput: %s", err, buf.String())
	}
	if info.Version == "" {
		t.Error("expected non-empty Version field")
	}
	if info.Platform == "" {
		t.Error("expected non-empty Platform field")
	}
	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion field")
	}
	if info.DefaultImage == "" {
		t.Error("expected non-empty DefaultImage field")
	}
}

func TestVersionCommandOutputInvalid(t *testing.T) {
	t.Parallel()
	streams, _ := newTestStreams()
	c := NewCommand(noopLogger{}, streams)
	c.SetArgs([]string{"-o", "xml"})
	err := c.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported output format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestVersionCommandDefaultOutput(t *testing.T) {
	t.Parallel()
	streams, buf := newTestStreams()
	c := NewCommand(noopLogger{}, streams)
	c.SetArgs([]string{})
	if err := c.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "kind v") {
		t.Errorf("default output should start with 'kind v', got: %q", out)
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name              string
		version           string
		versionPreRelease string
		gitCommit         string
		gitCommitCount    string
		want              string
	}{
		{
			name:              "With git commit count and with commit hash",
			version:           "v0.27.0",
			versionPreRelease: "alpha",
			gitCommit:         "mocked-hash",
			gitCommitCount:    "mocked-count",
			want:              "v0.27.0-alpha.mocked-count+mocked-hash",
		},
		{
			name:              "Without git commit count and and with hash",
			version:           "v0.27.0",
			versionPreRelease: "beta",
			gitCommit:         "mocked-hash",
			gitCommitCount:    "",
			want:              "v0.27.0-beta+mocked-hash",
		},
		{
			name:              "Without git commit hash and with commit count",
			version:           "v0.30.0",
			versionPreRelease: "alpha",
			gitCommit:         "",
			gitCommitCount:    "mocked-count",
			want:              "v0.30.0-alpha.mocked-count",
		},
		{
			name:              "Without git commit hash and without commit count",
			version:           "v0.27.0",
			versionPreRelease: "alpha",
			gitCommit:         "",
			gitCommitCount:    "",
			want:              "v0.27.0-alpha",
		},
		{
			name:              "Without pre release version",
			version:           "v0.27.0",
			versionPreRelease: "",
			gitCommit:         "",
			gitCommitCount:    "",
			want:              "v0.27.0",
		},
		{
			name:              "Without pre release version and with git commit hash and count",
			version:           "v0.27.0",
			versionPreRelease: "",
			gitCommit:         "mocked-commit",
			gitCommitCount:    "mocked-count",
			want:              "v0.27.0",
		},
	}
	for _, tt := range tests {
		// TODO: this won't be necessary when we require go 1.22+
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := version(tt.version, tt.versionPreRelease, tt.gitCommit, tt.gitCommitCount); got != tt.want {
				t.Errorf("Version() = %v, want %v", got, tt.want)
			}
		})
	}
}
