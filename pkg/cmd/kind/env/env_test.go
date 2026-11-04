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

package env

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintEnv(t *testing.T) {
	t.Setenv("KIND_CLUSTER_NAME", "kind-dev")
	t.Setenv("KIND_EXPERIMENTAL_PROVIDER", "podman")
	t.Setenv("KUBECONFIG", "/tmp/kubeconfig")
	t.Setenv("KIND_DNS_SEARCH", "")

	var buf bytes.Buffer
	if err := printEnv(&buf); err != nil {
		t.Fatalf("printEnv() returned error: %v", err)
	}
	out := buf.String()

	expected := []string{
		"NAME",
		"VALUE",
		"DESCRIPTION",
		"KIND_CLUSTER_NAME",
		"kind-dev",
		"KIND_EXPERIMENTAL_PROVIDER",
		"podman",
		"KUBECONFIG",
		"/tmp/kubeconfig",
		"KIND_DNS_SEARCH",
		"<empty>",
	}
	for _, want := range expected {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q did not contain %q", out, want)
		}
	}

	order := []string{
		"KIND_CLUSTER_NAME",
		"KIND_EXPERIMENTAL_PROVIDER",
		"KIND_EXPERIMENTAL_DOCKER_NETWORK",
		"KIND_EXPERIMENTAL_PODMAN_NETWORK",
		"KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER",
		"KIND_DNS_SEARCH",
		"KUBECONFIG",
	}
	last := -1
	for _, want := range order {
		idx := strings.Index(out, want)
		if idx < 0 {
			t.Fatalf("output %q did not contain %q", out, want)
		}
		if idx < last {
			t.Fatalf("env output order is wrong: %q appears before a previous entry", want)
		}
		last = idx
	}
}
