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
)

func TestLoadCurrent(t *testing.T) {
	cases := []struct {
		TestName       string
		Path           string
		ExpectReplicas []string
		ExpectError    bool
	}{
		{
			TestName:       "no config",
			Path:           "",
			ExpectReplicas: []string{"control-plane"}, // no config (empty config path) should return a single node cluster
			ExpectError:    false,
		},
		{
			TestName:       "v1alpha1 minimal",
			Path:           "./testdata/v1alpha1/valid-minimal.yaml",
			ExpectReplicas: []string{"control-plane"},
			ExpectError:    false,
		},
		{
			TestName:       "v1alpha1 with lifecyclehooks",
			Path:           "./testdata/v1alpha1/valid-with-lifecyclehooks.yaml",
			ExpectReplicas: []string{"control-plane"},
			ExpectError:    false,
		},
		{
			TestName:       "v1alpha2 minimal",
			Path:           "./testdata/v1alpha2/valid-minimal.yaml",
			ExpectReplicas: []string{"control-plane"},
			ExpectError:    false,
		},
		{
			TestName:       "v1alpha2 lifecyclehooks",
			Path:           "./testdata/v1alpha2/valid-with-lifecyclehooks.yaml",
			ExpectReplicas: []string{"control-plane"},
			ExpectError:    false,
		},
		{
			TestName:       "v1alpha2 config with 2 nodes",
			Path:           "./testdata/v1alpha2/valid-minimal-two-nodes.yaml",
			ExpectReplicas: []string{"control-plane", "worker"},
			ExpectError:    false,
		},
		{
			TestName:       "v1alpha2 full HA",
			Path:           "./testdata/v1alpha2/valid-full-ha.yaml",
			ExpectReplicas: []string{"etcd", "lb", "control-plane1", "control-plane2", "control-plane3", "worker1", "worker2"},
			ExpectError:    false,
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
		t.Run(c.TestName, func(t2 *testing.T) {
			// Loading config and deriving infos
			cfg, err := Load(c.Path)

			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t2.Fatalf("unexpected error while Loading config: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t2.Fatalf("unexpected lack or error while Loading config")
				}
			}

			if len(cfg.AllReplicas()) != len(c.ExpectReplicas) {
				t2.Fatalf("expected %d replicas, saw %d", len(c.ExpectReplicas), len(cfg.AllReplicas()))
			}

			for i, name := range c.ExpectReplicas {
				if cfg.AllReplicas()[i].Name != name {
					t2.Errorf("expected %q node at position %d, saw %q", name, i, cfg.AllReplicas()[i].Name)
				}
			}
		})
	}
}
