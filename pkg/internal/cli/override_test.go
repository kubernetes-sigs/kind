/*
Copyright 2026 The Kubernetes Authors.

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

package cli

import (
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/constants"
)

func TestNameFromEnv(t *testing.T) {
	t.Setenv(ClusterNameEnv, "env-cluster")

	if got := NameFromEnv(); got != "env-cluster" {
		t.Fatalf("NameFromEnv() = %q, want %q", got, "env-cluster")
	}
}

func TestDefaultName(t *testing.T) {
	t.Setenv(ClusterNameEnv, "")
	if got := DefaultName(); got != constants.DefaultClusterName {
		t.Fatalf("DefaultName() = %q, want %q", got, constants.DefaultClusterName)
	}

	t.Setenv(ClusterNameEnv, "env-cluster")
	if got := DefaultName(); got != "env-cluster" {
		t.Fatalf("DefaultName() = %q, want %q", got, "env-cluster")
	}
}
