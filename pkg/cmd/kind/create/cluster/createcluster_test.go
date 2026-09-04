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

package cluster

import (
	"testing"

	"sigs.k8s.io/kind/pkg/cmd"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/log"
)

func TestNameFlagDefaultPreservesConfigPrecedence(t *testing.T) {
	t.Setenv(cli.ClusterNameEnv, "")

	command := NewCommand(log.NoopLogger{}, cmd.IOStreams{})
	flag := command.Flags().Lookup("name")
	if flag == nil {
		t.Fatal("expected --name flag")
	}
	if flag.DefValue != "" {
		t.Fatalf("--name default = %q, want empty so config name can apply", flag.DefValue)
	}
	if got := flag.Value.String(); got != "" {
		t.Fatalf("--name value = %q, want empty so config name can apply", got)
	}
}

func TestNameFlagDefaultUsesEnv(t *testing.T) {
	t.Setenv(cli.ClusterNameEnv, "env-cluster")

	command := NewCommand(log.NoopLogger{}, cmd.IOStreams{})
	flag := command.Flags().Lookup("name")
	if flag == nil {
		t.Fatal("expected --name flag")
	}
	if flag.DefValue != "env-cluster" {
		t.Fatalf("--name default = %q, want %q", flag.DefValue, "env-cluster")
	}
	if got := flag.Value.String(); got != "env-cluster" {
		t.Fatalf("--name value = %q, want %q", got, "env-cluster")
	}

	if err := command.Flags().Set("name", "flag-cluster"); err != nil {
		t.Fatalf("setting --name: %v", err)
	}
	if got := flag.Value.String(); got != "flag-cluster" {
		t.Fatalf("--name value after explicit set = %q, want %q", got, "flag-cluster")
	}
}
