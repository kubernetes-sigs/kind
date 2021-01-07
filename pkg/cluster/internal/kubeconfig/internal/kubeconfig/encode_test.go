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
	"testing"

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestEncodeRoundtrip(t *testing.T) {
	t.Parallel()
	// test round tripping a kubeconfig
	const aConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: definitelyacert
    server: https://192.168.9.4:6443
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
    client-key-data: yup
`
	cfg, err := KINDFromRawKubeadm(aConfig, "kind", "")
	if err != nil {
		t.Fatalf("failed to decode kubeconfig: %v", err)
	}
	encoded, err := Encode(cfg)
	if err != nil {
		t.Fatalf("failed to encode kubeconfig: %v", err)
	}
	assert.StringEqual(t, aConfig, string(encoded))
}

func TestEncodeEmpty(t *testing.T) {
	t.Parallel()
	encoded, err := Encode(&Config{})
	if err != nil {
		t.Fatalf("failed to encode kubeconfig: %v", err)
	}
	assert.StringEqual(t, "", string(encoded))
}
