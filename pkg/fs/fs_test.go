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

// Package fs contains utilities for interacting with the host filesystem
// in a docker friendly way
// TODO(bentheelder): this should be internal

package fs

import (
	"os"
	"testing"
)

func TestIsAbs(t *testing.T) {
	cases := []struct {
        name     string
        path     string
        expected bool
    }{
        {
            name: "linux absolute path",
            path: "/usr/local/bin",
            expected: true,
        },
        {
            name: "relative path",
            path:     "usr/local/bin",
            expected: false,
        },
        {
            name: "relative path with dot",
            path: "./usr/local/bin",
            expected: false,
        },
        {
            name: "empty path",
            path: "",
            expected: false,
        },
    }

	for _, tcases := range cases {
		t.Run(tcases.name, func(t *testing.T) {
			exists := IsAbs(tcases.path)

			if exists != tcases.expected {
				t.Errorf("Wanted IsAbs(%q) to be %v, got %v", tcases.path, tcases.expected, exists)
			}
		})
	}
}

func TestTempDir(t *testing.T) {
	dir, err := TempDir("", "kind-fs-test-")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}

	defer os.RemoveAll(dir) // Cleaning up after the test finishes

	info, err := os.Stat(dir)

	if err != nil {
		t.Fatalf("failed to test stat temp dir: %v", err)
	}
	if !info.IsDir() { // if the file created info is not a directory.
		t.Errorf("Expected %q to be in the directory", dir)
	}
}