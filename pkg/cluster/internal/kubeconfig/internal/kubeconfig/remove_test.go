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
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestRemove(t *testing.T) {
	t.Parallel()
	cases := []struct {
		Name           string
		Existing       *Config
		ClusterName    string
		Expected       *Config
		ExpectModified bool
	}{
		{
			Name:           "empty config",
			Existing:       &Config{},
			ClusterName:    "foo",
			Expected:       &Config{},
			ExpectModified: false,
		},
		{
			Name: "remove kind from only kind",
			Existing: &Config{
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
			ClusterName: "kind",
			Expected: &Config{
				Clusters: []NamedCluster{},
				Users:    []NamedUser{},
				Contexts: []NamedContext{},
			},
			ExpectModified: true,
		},
		{
			Name: "remove kind, leave kops",
			Existing: &Config{
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
				CurrentContext: "kind-kind",
			},
			ClusterName: "kind",
			Expected: &Config{
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
				CurrentContext: "",
			},
			ExpectModified: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			modified := remove(tc.Existing, tc.ClusterName)
			if modified != tc.ExpectModified {
				if tc.ExpectModified {
					t.Errorf("Expected config to be modified but got modified == false")
				} else {
					t.Errorf("Expected config to be modified but got modified == true")
				}
			}
			assert.DeepEqual(t, tc.Expected, tc.Existing)
		})
	}
}

func TestRemoveKIND(t *testing.T) {
	t.Parallel()
	t.Run("only kind", testRemoveKINDTrivial)
	t.Run("leave another cluster", testRemoveKINDKeepOther)
}

func testRemoveKINDTrivial(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "kind-testremovekind")
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

	// ensure that we can write this merged config
	if err := RemoveKIND("foo", existingConfigPath); err != nil {
		t.Fatalf("Failed to remove kind from kubeconfig: %v", err)
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
kind: Config
preferences: {}
`
	assert.StringEqual(t, expected, string(contents))
}

func testRemoveKINDKeepOther(t *testing.T) {
	// tests removing a kind cluster but keeping another cluster
	t.Parallel()
	dir, err := os.MkdirTemp("", "kind-testremovekind")
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
- cluster:
    certificate-authority-data: definitelyacert
    server: https://192.168.9.4:6443
  name: kops-foo
contexts:
- context:
    cluster: kind-foo
    user: kind-foo
  name: kind-foo
- context:
    cluster: kops-foo
    user: kops-foo
  name: kops-foo
current-context: kops-foo
kind: Config
apiVersion: v1
preferences: {}
users:
- name: kind-foo
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
- name: kops-foo
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
`
	existingConfigPath := filepath.Join(dir, "existing-kubeconfig")
	if err := os.WriteFile(existingConfigPath, []byte(existingConfig), os.ModePerm); err != nil {
		t.Fatalf("Failed to create existing kubeconfig: %d", err)
	}

	// ensure that we can write this merged config
	if err := RemoveKIND("foo", existingConfigPath); err != nil {
		t.Fatalf("Failed to remove kind from kubeconfig: %v", err)
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
  name: kops-foo
contexts:
- context:
    cluster: kops-foo
    user: kops-foo
  name: kops-foo
current-context: kops-foo
kind: Config
preferences: {}
users:
- name: kops-foo
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
`
	assert.StringEqual(t, expected, string(contents))
}
