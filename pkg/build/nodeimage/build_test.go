/*
Copyright The Kubernetes Authors.

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

package nodeimage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBuildType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kind-build-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "some-file")
	if err := os.WriteFile(tmpFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	cases := []struct {
		name     string
		param    string
		expected string
	}{
		{
			name:     "valid http url",
			param:    "http://example.com/foo",
			expected: "url",
		},
		{
			name:     "valid https url",
			param:    "https://example.com/foo",
			expected: "url",
		},
		{
			name:     "existing file",
			param:    tmpFile,
			expected: "file",
		},
		{
			name:     "existing directory",
			param:    tmpDir,
			expected: "source",
		},
		{
			name:     "ci/latest build type",
			param:    "ci/latest",
			expected: "ci",
		},
		{
			name:     "ci/latest-1.30 build type",
			param:    "ci/latest-1.30",
			expected: "ci",
		},
		{
			name:     "valid semantic release",
			param:    "v1.30.0",
			expected: "release",
		},
		{
			name:     "valid semantic release without v prefix",
			param:    "1.30.0",
			expected: "release",
		},
		{
			name:     "invalid / fallback",
			param:    "invalid-input",
			expected: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual := detectBuildType(tc.param)
			if actual != tc.expected {
				t.Errorf("detectBuildType(%q) = %q, expected %q", tc.param, actual, tc.expected)
			}
		})
	}
}
