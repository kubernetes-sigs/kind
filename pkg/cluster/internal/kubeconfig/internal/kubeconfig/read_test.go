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
	"reflect"
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestKINDFromRawKubeadm(t *testing.T) {
	t.Parallel()
	// test that a bogus config is caught
	t.Run("bad config", func(t *testing.T) {
		t.Parallel()
		_, err := KINDFromRawKubeadm("	", "kind", "")
		assert.ExpectError(t, true, err)
	})
	// test reading a legitimate kubeadm config and converting it to a kind config
	t.Run("valid config", func(t *testing.T) {
		const rawConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: definitelyacert
    server: https://192.168.9.4:6443
  name: kind
contexts:
- context:
    cluster: kind
    user: kubernetes-admin
  name: kubernetes-admin@kind
current-context: kubernetes-admin@kind
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    client-certificate-data: seemslegit
    client-key-data: yep
`
		server := "https://127.0.0.1:6443"
		expected := &Config{
			Clusters: []NamedCluster{
				{
					Name: "kind-kind",
					Cluster: Cluster{
						Server: server,
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
		cfg, err := KINDFromRawKubeadm(rawConfig, "kind", server)
		if err != nil {
			t.Fatalf("failed to decode kubeconfig: %v", err)
		}
		if !reflect.DeepEqual(cfg, expected) {
			t.Errorf("Read Config did not equal Expected")
			t.Errorf("Expected: %+v", expected)
			t.Errorf("Actual: %+v", cfg)
			t.Errorf("type: %s", reflect.TypeOf(cfg.OtherFields["preferences"]))
		}
	})
}
