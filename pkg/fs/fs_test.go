/*
Copyright 2020 The Kubernetes Authors.

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

package fs

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestTempDir(t *testing.T) {
	tempDir := os.TempDir()
	tests := []struct {
		name          string
		dir           string
		prefix        string
		expectedError bool
	}{
		{
			name:          "all is right",
			dir:           tempDir,
			prefix:        "tmp",
			expectedError: false,
		},
		{
			name:          "dir is empty",
			dir:           "",
			prefix:        "tmp",
			expectedError: false,
		},
		{
			name:          "dir doesn't exist",
			dir:           "/abc",
			prefix:        "tmp",
			expectedError: true,
		},
		{
			name:          "prefix is empty",
			dir:           tempDir,
			prefix:        "",
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := TempDir(tt.dir, tt.prefix)
			os.Remove(name)
			if (nil != err) != tt.expectedError {
				t.Errorf("expectedError: %v, got: %v", tt.expectedError, err)
			}
		})
	}
}

func TestIsAbs(t *testing.T) {
	tests := []struct {
		name     string
		hostPath string
		expected bool
	}{
		{
			name:     "is absolute path",
			hostPath: "/home/test",
			expected: true,
		},
		{
			name:     "is relative path",
			hostPath: "./test",
			expected: false,
		},
		{
			name:     "is a home path",
			hostPath: "~/test",
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := IsAbs(tt.hostPath)
			if ret != tt.expected {
				t.Errorf("expected: %v, got: %v", tt.expected, ret)
			}
		})
	}
}

func TestCopy(t *testing.T) {
	testDir, err := ioutil.TempDir(os.TempDir(), "")
	os.Mkdir(testDir+"/td1", 0777)
	if nil != err {
		return
	}
	defer os.Remove(testDir)

	tests := []struct {
		name          string
		src           string
		dst           string
		expectedError bool
	}{
		{
			name:          "src is empty",
			src:           "",
			dst:           testDir + "/test.txt",
			expectedError: true,
		},
		{
			name:          "src doesn't exist",
			src:           testDir + "/abc",
			dst:           testDir + "/test.txt",
			expectedError: true,
		},
		// src is dir
		{
			name:          "all is right",
			src:           testDir + "/td1",
			dst:           testDir + "/test",
			expectedError: false,
		},
		{
			name:          "dst is empty",
			src:           testDir + "/td1",
			dst:           "",
			expectedError: true,
		},
		// src is file
		{
			name:          "all is right",
			src:           "fs_test.go",
			dst:           testDir + "/test.txt",
			expectedError: false,
		},
		{
			name:          "dst is dir",
			src:           "fs_test.go",
			dst:           testDir + "/td1",
			expectedError: true,
		},
		{
			name:          "dst is empty",
			src:           "fs_test.go",
			dst:           "",
			expectedError: true,
		},
		{
			name:          "dst is exist",
			src:           "fs_test.go",
			dst:           testDir + "/test.txt",
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Copy(tt.src, tt.dst)
			if (nil != err) != tt.expectedError {
				t.Errorf("expectedError: %v, got: %v", tt.expectedError, err)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	testDir, err := ioutil.TempDir(os.TempDir(), "")
	os.Mkdir(testDir+"/td1", 0777)
	if nil != err {
		return
	}
	defer os.Remove(testDir)

	tests := []struct {
		name          string
		src           string
		dst           string
		expectedError bool
	}{
		{
			name:          "all is right",
			src:           "fs_test.go",
			dst:           testDir + "/test.txt",
			expectedError: false,
		},
		{
			name:          "src is dir",
			src:           testDir + "/td1",
			dst:           testDir + "/test.txt",
			expectedError: true,
		},
		{
			name:          "src is empty",
			src:           "",
			dst:           testDir + "/test.txt",
			expectedError: true,
		},
		{
			name:          "src doesn't exist",
			src:           "abc.go",
			dst:           testDir + "/test.txt",
			expectedError: true,
		},
		{
			name:          "dst is dir",
			src:           "fs_test.go",
			dst:           testDir + "/td1",
			expectedError: true,
		},
		{
			name:          "dst is empty",
			src:           "fs_test.go",
			dst:           "",
			expectedError: true,
		},
		{
			name:          "dst is exist",
			src:           "fs_test.go",
			dst:           testDir + "/test.txt",
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyFile(tt.src, tt.dst)
			if (nil != err) != tt.expectedError {
				t.Errorf("expectedError: %v, got: %v", tt.expectedError, err)
			}
		})
	}
}
