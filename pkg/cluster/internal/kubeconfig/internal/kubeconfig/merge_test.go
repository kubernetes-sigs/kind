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
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestMerge(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name        string
		Existing    *Config
		Kind        *Config
		Expected    *Config
		ExpectError bool
	}{
		{
			Name:        "bad kind config",
			Existing:    &Config{},
			Kind:        &Config{},
			Expected:    &Config{},
			ExpectError: true,
		},
		{
			Name:     "empty existing",
			Existing: &Config{},
			Kind: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Expected: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			ExpectError: false,
		},
		{
			Name: "replace existing",
			Existing: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
						Cluster: Cluster{
							Server: "foo",
						},
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Kind: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Expected: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			ExpectError: false,
		},
		{
			Name: "add to existing",
			Existing: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kops-blah",
						Cluster: Cluster{
							Server: "foo",
						},
					},
				},
				Users: []NamedUser{
					{
						Name: "kops-blah",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kops-blah",
					},
				},
			},
			Kind: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kind-kind",
					},
				},
			},
			Expected: &Config{
				Clusters: []NamedCluster{
					{
						Name: "kops-blah",
						Cluster: Cluster{
							Server: "foo",
						},
					},
					{
						Name: "kind-kind",
					},
				},
				Users: []NamedUser{
					{
						Name: "kops-blah",
					},
					{
						Name: "kind-kind",
					},
				},
				Contexts: []NamedContext{
					{
						Name: "kops-blah",
					},
					{
						Name: "kind-kind",
					},
				},
			},
			ExpectError: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			err := merge(tc.Existing, tc.Kind)
			assert.ExpectError(t, tc.ExpectError, err)
			if !tc.ExpectError && !reflect.DeepEqual(tc.Existing, tc.Expected) {
				t.Errorf("Merged Config did not equal Expected")
				t.Errorf("Expected: %+v", tc.Expected)
				t.Errorf("Actual: %+v", tc.Existing)
			}
		})
	}
}

func TestWriteMerged(t *testing.T) {
	t.Parallel()
	t.Run("normal merge", testWriteMergedNormal)
	t.Run("bad kind config", testWriteMergedBogusConfig)
	t.Run("merge into non-existent file", testWriteMergedNoExistingFile)
}

func testWriteMergedNormal(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "kind-testwritemerged")
	if err != nil {
		t.Fatalf("Failed to create tempdir: %d", err)
	}
	defer os.RemoveAll(dir)

	// create an existing kubeconfig
	const existingConfig = `clusters:
- cluster:
    certificate-authority-data: definitelyacert
    server: https://192.168.9.4:6443
  name: kind-foo
contexts:
- context:
    cluster: kind-foo
    user: kind-foo
  name: kind-foo
current-context: kind-foo
kind: Config
apiVersion: v1
preferences: {}
users:
- name: kind-foo
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
`
	existingConfigPath := filepath.Join(dir, "existing-kubeconfig")
	if err := os.WriteFile(existingConfigPath, []byte(existingConfig), os.ModePerm); err != nil {
		t.Fatalf("Failed to create existing kubeconfig: %d", err)
	}

	kindConfig := &Config{
		Clusters: []NamedCluster{
			{
				Name: "kind-kind",
				Cluster: Cluster{
					Server: "https://127.0.0.1:6443",
					OtherFields: map[string]interface{}{
						"certificate-authority-data": "definitelyacert",
					},
				},
			},
		},
		Contexts: []NamedContext{
			{
				Name: "kind-kind",
				Context: Context{
					User:    "kind-kind",
					Cluster: "kind-kind",
				},
			},
		},
		Users: []NamedUser{
			{
				Name: "kind-kind",
				User: map[string]interface{}{
					"client-certificate-data": "seemslegit",
					"client-key-data":         "yep",
				},
			},
		},
		CurrentContext: "kind-kind",
		OtherFields: map[string]interface{}{
			"apiVersion":  "v1",
			"kind":        "Config",
			"preferences": map[string]interface{}{},
		},
	}
	// ensure that we can write this merged config
	if err := WriteMerged(kindConfig, existingConfigPath); err != nil {
		t.Fatalf("Failed to write merged kubeconfig: %v", err)
	}

	// ensure the output matches expected
	f, err := os.Open(existingConfigPath)
	if err != nil {
		t.Fatalf("Failed to open merged kubeconfig: %v", err)
	}
	contents, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("Failed to read merged kubeconfig: %v", err)
	}
	expected := `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: definitelyacert
    server: https://192.168.9.4:6443
  name: kind-foo
- cluster:
    certificate-authority-data: definitelyacert
    server: https://127.0.0.1:6443
  name: kind-kind
contexts:
- context:
    cluster: kind-foo
    user: kind-foo
  name: kind-foo
- context:
    cluster: kind-kind
    user: kind-kind
  name: kind-kind
current-context: kind-kind
kind: Config
preferences: {}
users:
- name: kind-foo
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
- name: kind-kind
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
`
	assert.StringEqual(t, expected, string(contents))
}

func testWriteMergedBogusConfig(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "kind-testwritemerged")
	if err != nil {
		t.Fatalf("Failed to create tempdir: %d", err)
	}
	defer os.RemoveAll(dir)

	err = WriteMerged(&Config{}, filepath.Join(dir, "bogus"))
	assert.ExpectError(t, true, err)
}

func testWriteMergedNoExistingFile(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "kind-testwritemerged")
	if err != nil {
		t.Fatalf("Failed to create tempdir: %d", err)
	}
	defer os.RemoveAll(dir)

	kindConfig := &Config{
		Clusters: []NamedCluster{
			{
				Name: "kind-kind",
				Cluster: Cluster{
					Server: "https://127.0.0.1:6443",
					OtherFields: map[string]interface{}{
						"certificate-authority-data": "definitelyacert",
					},
				},
			},
		},
		Contexts: []NamedContext{
			{
				Name: "kind-kind",
				Context: Context{
					User:    "kind-kind",
					Cluster: "kind-kind",
				},
			},
		},
		Users: []NamedUser{
			{
				Name: "kind-kind",
				User: map[string]interface{}{
					"client-certificate-data": "seemslegit",
					"client-key-data":         "yep",
				},
			},
		},
		CurrentContext: "kind-kind",
		OtherFields: map[string]interface{}{
			"apiVersion":  "v1",
			"kind":        "Config",
			"preferences": map[string]interface{}{},
		},
	}

	nonExistentPath := filepath.Join(dir, "bogus", "extra-bogus")
	err = WriteMerged(kindConfig, nonExistentPath)
	assert.ExpectError(t, false, err)

	// ensure the output matches expected
	f, err := os.Open(nonExistentPath)
	if err != nil {
		t.Fatalf("Failed to open merged kubeconfig: %v", err)
	}
	contents, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("Failed to read merged kubeconfig: %v", err)
	}
	expected := `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: definitelyacert
    server: https://127.0.0.1:6443
  name: kind-kind
contexts:
- context:
    cluster: kind-kind
    user: kind-kind
  name: kind-kind
current-context: kind-kind
kind: Config
preferences: {}
users:
- name: kind-kind
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
`
	assert.StringEqual(t, expected, string(contents))
}
