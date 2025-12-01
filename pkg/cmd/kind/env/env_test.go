/*
Copyright 2024 The Kubernetes Authors.

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
	"fmt"
	"io"
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/log"
)

func TestEnvCommandOutputsVariables(t *testing.T) {
	testValues := map[string]string{
		"KIND_EXPERIMENTAL_PROVIDER":               "podman",
		"KIND_CLUSTER_NAME":                        "test-cluster",
		"KUBECONFIG":                               "/tmp/kubeconfig",
		"KIND_EXPERIMENTAL_DOCKER_NETWORK":         "kind-network",
		"KIND_EXPERIMENTAL_PODMAN_NETWORK":         "podman-network",
		"KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER": "fuse-overlayfs",
	}
	for key, value := range testValues {
		t.Setenv(key, value)
	}

	out := &bytes.Buffer{}
	streams := cmd.IOStreams{
		Out:    out,
		ErrOut: io.Discard,
	}
	command := NewCommand(log.NoopLogger{}, streams)
	command.SetArgs([]string{})

	if err := command.Execute(); err != nil {
		t.Fatalf("env command returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != len(envVariables) {
		t.Fatalf("expected %d environment variables, got %d", len(envVariables), len(lines))
	}

	for i, variable := range envVariables {
		parts := strings.SplitN(lines[i], "\t", 2)
		if len(parts) != 2 {
			t.Fatalf("line %d not formatted correctly: %q", i, lines[i])
		}

		value := testValues[variable.Name]
		expectedPrefix := fmt.Sprintf("%s=%q", variable.Name, value)
		if parts[0] != expectedPrefix {
			t.Errorf("expected %q, got %q", expectedPrefix, parts[0])
		}
		if parts[1] != variable.Description {
			t.Errorf("expected description %q, got %q", variable.Description, parts[1])
		}
	}
}
