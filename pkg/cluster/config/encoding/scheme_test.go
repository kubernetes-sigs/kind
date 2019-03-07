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

package encoding

import (
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/config"
)

func TestNewConfig(t *testing.T) {
	cases := []struct {
		TestName                 string
		Path                     string
		ControlPlaneNodes        int32
		WorkerNodes              int32
		ExternaloadBalancerNodes int32
		ExpectError              bool
	}{
		{
			TestName:          "Defaults",
			Path:              "",
			ControlPlaneNodes: 1,
			ExpectError:       false,
		},
		{
			TestName:          "1 ControlPlaneNodes + n WorkerNodes",
			Path:              "",
			ControlPlaneNodes: 1,
			WorkerNodes:       2,
			ExpectError:       false,
		},
		{
			TestName:                 "n ControlPlaneNodes",
			Path:                     "",
			ControlPlaneNodes:        2,
			ExternaloadBalancerNodes: 1,
			ExpectError:              false,
		},
	}
	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			cfg, err := NewConfig(c.ControlPlaneNodes, c.WorkerNodes)

			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t.Fatalf("unexpected error while Loading config: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t.Fatalf("unexpected lack or error while Loading config")
				}
			}

			// check the expected nodes are there
			assertConfigHasNodes(t, cfg, config.ControlPlaneRole, c.ControlPlaneNodes)
			assertConfigHasNodes(t, cfg, config.ExternalLoadBalancerRole, c.ExternaloadBalancerNodes)
			assertConfigHasNodes(t, cfg, config.WorkerRole, c.WorkerNodes)
		})
	}
}

func TestLoadCurrent(t *testing.T) {
	cases := []struct {
		TestName                       string
		Path                           string
		ExpectControlPlaneNodes        int32
		ExpectWorkerNodes              int32
		ExpectExternaloadBalancerNodes int32
		ExpectExternaEtcdNodes         int32
		ExpectError                    bool
	}{
		{
			TestName:                "v1alpha1 minimal",
			Path:                    "./testdata/v1alpha1/valid-minimal.yaml",
			ExpectControlPlaneNodes: 1,
			ExpectError:             false,
		},
		{
			TestName:                "v1alpha1 with lifecyclehooks",
			Path:                    "./testdata/v1alpha1/valid-with-lifecyclehooks.yaml",
			ExpectControlPlaneNodes: 1,
			ExpectError:             false,
		},
		{
			TestName:                "v1alpha2 minimal",
			Path:                    "./testdata/v1alpha2/valid-minimal.yaml",
			ExpectControlPlaneNodes: 1,
			ExpectError:             false,
		},
		{
			TestName:                "v1alpha2 config with 2 nodes",
			Path:                    "./testdata/v1alpha2/valid-minimal-two-nodes.yaml",
			ExpectControlPlaneNodes: 1,
			ExpectWorkerNodes:       1,
			ExpectError:             false,
		},
		{
			TestName:                       "v1alpha2 full HA",
			Path:                           "./testdata/v1alpha2/valid-full-ha.yaml",
			ExpectExternaEtcdNodes:         1,
			ExpectExternaloadBalancerNodes: 1,
			ExpectControlPlaneNodes:        3,
			ExpectWorkerNodes:              2,
			ExpectError:                    false,
		},
		{
			TestName:    "invalid path",
			Path:        "./testdata/not-a-file.bogus",
			ExpectError: true,
		},
		{
			TestName:    "Invalid apiversion",
			Path:        "./testdata/invalid-apiversion.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid kind",
			Path:        "./testdata/invalid-kind.yaml",
			ExpectError: true,
		},
		{
			TestName:    "Invalid yaml",
			Path:        "./testdata/invalid-yaml.yaml",
			ExpectError: true,
		},
	}
	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			cfg, err := Load(c.Path)

			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t.Fatalf("unexpected error while Loading config: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t.Fatalf("unexpected lack or error while Loading config")
				}
			}

			assertConfigHasNodes(t, cfg, config.ExternalLoadBalancerRole, c.ExpectExternaloadBalancerNodes)
			assertConfigHasNodes(t, cfg, config.ExternalEtcdRole, c.ExpectExternaEtcdNodes)
			assertConfigHasNodes(t, cfg, config.ControlPlaneRole, c.ExpectControlPlaneNodes)
			assertConfigHasNodes(t, cfg, config.WorkerRole, c.ExpectWorkerNodes)
		})
	}
}

func assertConfigHasNodes(t *testing.T, cfg *config.Config, role config.NodeRole, expectedReplicas int32) {
	for _, n := range cfg.Nodes {
		if n.Role != role {
			continue
		}

		var actualReplicas int32 = 1
		if n.Replicas != nil {
			actualReplicas = *n.Replicas
		}

		if actualReplicas == expectedReplicas {
			return
		}

		t.Fatalf("expected %d replicas with role %s, saw %d", expectedReplicas, role, actualReplicas)
	}

	if expectedReplicas != 0 {
		t.Fatalf("config does not have nodes with role %s", role)
	}
}
