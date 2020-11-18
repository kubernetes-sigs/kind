// +build !nointegration

/*
Copyright 2020 The Kubernetes Authors.

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

package docker

import (
	"fmt"
	"regexp"
	"testing"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/internal/integration"
)

func TestIntegrationEnsureNetworkConcurrent(t *testing.T) {
	integration.MaybeSkip(t)

	testNetworkName := "integration-test-ensure-kind-network"

	// cleanup
	cleanup := func() {
		ids, _ := networksWithName(testNetworkName)
		if len(ids) > 0 {
			_ = deleteNetworks(ids...)
		}
	}
	cleanup()
	defer cleanup()

	// this is more than enough to trigger race conditions
	networkConcurrency := 10

	// Create multiple networks concurrently
	errCh := make(chan error, networkConcurrency)
	for i := 0; i < networkConcurrency; i++ {
		go func() {
			errCh <- ensureNetwork(testNetworkName)
		}()
	}
	for i := 0; i < networkConcurrency; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("error creating network: %v", err)
			rerr := exec.RunErrorForError(err)
			if rerr != nil {
				t.Errorf("%q", rerr.Output)
			}
			t.Errorf("%+v", errors.StackTrace(err))
		}
	}

	cmd := exec.Command(
		"docker", "network", "ls",
		fmt.Sprintf("--filter=name=^%s$", regexp.QuoteMeta(testNetworkName)),
		"--format={{.Name}}",
	)

	lines, err := exec.OutputLines(cmd)
	if err != nil {
		t.Errorf("obtaining the docker networks")
	}
	if len(lines) != 1 {
		t.Errorf("wrong number of networks created: %d", len(lines))
	}
}
