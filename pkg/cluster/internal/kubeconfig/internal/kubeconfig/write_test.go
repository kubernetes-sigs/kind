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

func TestWrite(t *testing.T) {
	t.Parallel()
	t.Run("non-existent file", testWriteNoExistingFile)
}

func testWriteNoExistingFile(t *testing.T) {
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
	err = write(kindConfig, nonExistentPath)
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
