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

package kubeconfig

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/fs"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestPaths(t *testing.T) {
	t.Parallel()
	// test explicit kubeconfig
	t.Run("explicit path", func(t *testing.T) {
		t.Parallel()
		explicitPath := "foo"
		result := paths(explicitPath, func(s string) string {
			return map[string]string{
				"KUBECONFIG": strings.Join([]string{"/foo", "/bar", "", "/foo", "/bar"}, string(filepath.ListSeparator)),
				"HOME":       "/home",
			}[s]
		})
		expected := []string{explicitPath}
		assert.DeepEqual(t, expected, result)
	})
	t.Run("KUBECONFIG list", func(t *testing.T) {
		t.Parallel()
		result := paths("", func(s string) string {
			return map[string]string{
				"KUBECONFIG": strings.Join([]string{"/foo", "/bar", "", "/foo", "/bar"}, string(filepath.ListSeparator)),
				"HOME":       "/home",
			}[s]
		})
		expected := []string{"/foo", "/bar"}
		assert.DeepEqual(t, expected, result)
	})
	t.Run("$HOME/.kube/config", func(t *testing.T) {
		t.Parallel()
		result := paths("", func(s string) string {
			return map[string]string{
				"HOME": "/home",
			}[s]
		})
		expected := []string{"/home/.kube/config"}
		assert.DeepEqual(t, expected, result)
	})
}

func TestPathForMerge(t *testing.T) {
	t.Parallel()
	// create a directory structure with some files to be "kubeconfigs"
	dir, err := fs.TempDir("", "kind-testwritemerged")
	if err != nil {
		t.Fatalf("Failed to create tempdir: %v", err)
	}
	defer os.RemoveAll(dir)

	// create a fake homedir
	homeDir := filepath.Join(dir, "fake-home")
	if err := os.Mkdir(homeDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create fake home dir: %v", err)
	}

	// create some fake KUBECONFIG files
	fakeKubeconfigs := []string{}
	for _, partialPath := range []string{"foo", "bar", "baz"} {
		p := filepath.Join(dir, partialPath)
		fakeKubeconfigs = append(fakeKubeconfigs, p)
		f, err := os.Create(p)
		if err != nil {
			t.Fatalf("failed to create fake kubeconfig file: %v", err)
		}
		f.Close()
	}

	// test explicit kubeconfig
	t.Run("explicit path", func(t *testing.T) {
		explicitPath := "foo"
		result := pathForMerge(explicitPath, func(s string) string {
			return map[string]string{
				"KUBECONFIG": strings.Join([]string{"/foo", "/bar", "", "/foo", "/bar"}, string(filepath.ListSeparator)),
				"HOME":       "/home",
			}[s]
		})
		expected := explicitPath
		assert.StringEqual(t, expected, result)
	})
	t.Run("KUBECONFIG list", func(t *testing.T) {
		result := pathForMerge("", func(s string) string {
			return map[string]string{
				"KUBECONFIG": strings.Join(fakeKubeconfigs, string(filepath.ListSeparator)),
			}[s]
		})
		expected := fakeKubeconfigs[0]
		assert.StringEqual(t, expected, result)
	})
	t.Run("KUBECONFIG select last if none exist", func(t *testing.T) {
		kubeconfigEnvValue := strings.Join([]string{"/bogus/path", "/bogus/path/two"}, string(filepath.ListSeparator))
		result := pathForMerge("", func(s string) string {
			return map[string]string{
				"KUBECONFIG": kubeconfigEnvValue,
			}[s]
		})
		expected := "/bogus/path/two"
		assert.StringEqual(t, expected, result)
	})
}

func TestHomeDir(t *testing.T) {
	t.Parallel()
	t.Run("windows HOME with .kube/config", func(t *testing.T) {
		t.Parallel()
		// create a directory structure with a "kubeconfigs"
		dir, err := fs.TempDir("", "kind-testwritemerged")
		if err != nil {
			t.Fatalf("Failed to create tempdir: %v", err)
		}
		defer os.RemoveAll(dir)

		// create the fake kubeconfig
		fakeHomeDir := path.Join(dir, "fake-home")
		fakeKubeConfig := path.Join(fakeHomeDir, ".kube", "config")
		if err := os.MkdirAll(path.Dir(fakeKubeConfig), os.ModePerm); err != nil {
			t.Fatalf("Failed to create fake kubeconfig dir: %v", err)
		}
		f, err := os.Create(fakeKubeConfig)
		if err != nil {
			t.Fatalf("Failed to create tempdir: %v", err)
		}
		f.Close()

		// this should return the fake kubeconfig
		result := homeDir("windows", func(e string) string {
			return map[string]string{
				"HOME":      fakeHomeDir,
				"HOMEDRIVE": "ZZ:",
				"HOMEPATH":  `ZZ:\Users\fake-user-zzz`,
			}[e]
		})
		assert.StringEqual(t, fakeHomeDir, result)
	})
	t.Run("windows HOME without .kube/config", func(t *testing.T) {
		t.Parallel()
		// create a fake home dir
		fakeHomeDir, err := fs.TempDir("", "kind-testwritemerged")
		if err != nil {
			t.Fatalf("Failed to create tempdir: %v", err)
		}
		defer os.RemoveAll(fakeHomeDir)

		// this should return the fake kubeconfig
		result := homeDir("windows", func(e string) string {
			return map[string]string{
				"HOME":      fakeHomeDir,
				"HOMEDRIVE": filepath.VolumeName(fakeHomeDir),
				"HOMEPATH":  path.Join("Users", "fake-user-zzz"),
			}[e]
		})
		assert.StringEqual(t, fakeHomeDir, result)
	})
	t.Run("windows HOME none exist", func(t *testing.T) {
		t.Parallel()
		// this should return the fake kubeconfig
		result := homeDir("windows", func(e string) string {
			return map[string]string{
				"HOME":      "Z:/faaaaake",
				"HOMEDRIVE": "Z:/",
				"HOMEPATH":  path.Join("Users", "fake-user-zzz"),
			}[e]
		})
		assert.StringEqual(t, "Z:/faaaaake", result)
	})
	t.Run("windows no path", func(t *testing.T) {
		t.Parallel()
		// this should return the fake kubeconfig
		result := homeDir("windows", func(e string) string {
			return ""
		})
		assert.StringEqual(t, "", result)
	})
}
