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

package config

import (
	"testing"

	"sigs.k8s.io/kind/pkg/util"
)

// TODO(fabriziopandini): ideally this should use scheme.Default, but this creates a circular dependency
// So the current solution is to mimic defaulting for the validation test
func newDefaultedNode(role NodeRole) Node {
	return Node{
		Role:  role,
		Image: "myImage:latest",
	}
}

func TestNodeValidate(t *testing.T) {
	cases := []struct {
		TestName     string
		Node         Node
		ExpectErrors int
	}{
		{
			TestName:     "Canonical node",
			Node:         newDefaultedNode(ControlPlaneRole),
			ExpectErrors: 0,
		},
		{
			TestName: "Invalid PreBoot hook",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.ControlPlane = &ControlPlane{
					NodeLifecycle: &NodeLifecycle{
						PreBoot: []LifecycleHook{
							{
								Command: []string{},
							},
						},
					},
				}
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Invalid PreKubeadm hook",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.ControlPlane = &ControlPlane{
					NodeLifecycle: &NodeLifecycle{
						PreKubeadm: []LifecycleHook{
							{
								Name:    "pull an image",
								Command: []string{},
							},
						},
					},
				}
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Invalid PostKubeadm hook",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.ControlPlane = &ControlPlane{
					NodeLifecycle: &NodeLifecycle{
						PostKubeadm: []LifecycleHook{
							{
								Name:    "pull an image",
								Command: []string{},
							},
						},
					},
				}
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Empty image field",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.Image = ""
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Empty role field",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.Role = ""
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Unknows role field",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.Role = "ssss"
				return cfg
			}(),
			ExpectErrors: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.TestName, func(t2 *testing.T) {
			err := tc.Node.Validate()
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil {
				if tc.ExpectErrors != 0 {
					t2.Error("received no errors but expected errors for case")
				}
				return
			}
			// - not castable to *Errors, in which case we have the wrong error type ...
			configErrors, ok := err.(util.Errors)
			if !ok {
				t2.Errorf("config.Validate should only return nil or ConfigErrors{...}, got: %v", err)
				return
			}
			// - ConfigErrors, in which case expect a certain number of errors
			errors := configErrors.Errors()
			if len(errors) != tc.ExpectErrors {
				t2.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errors, len(errors))
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		TestName     string
		Nodes        []Node
		ExpectErrors int
	}{
		{
			TestName: "Canonical config",
			Nodes: []Node{
				newDefaultedNode(ControlPlaneRole),
			},
		},
		{
			TestName:     "Fail without at least one control plane",
			ExpectErrors: 1,
		},
		{
			TestName: "Fail without at load balancer and more than one control plane",
			Nodes: []Node{
				newDefaultedNode(ControlPlaneRole),
				newDefaultedNode(ControlPlaneRole),
			},
			ExpectErrors: 1,
		},
		{
			TestName: "Fail with not valid nodes",
			Nodes: []Node{
				func() Node {
					cfg := newDefaultedNode(ControlPlaneRole)
					cfg.Image = ""
					return cfg
				}(),
				func() Node {
					cfg := newDefaultedNode(ControlPlaneRole)
					cfg.Role = ""
					return cfg
				}(),
			},
			ExpectErrors: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.TestName, func(t2 *testing.T) {
			var c = Config{Nodes: tc.Nodes}
			if err := c.DeriveInfo(); err != nil {
				t.Fatalf("unexpected error while adding nodes: %v", err)
			}

			// validating config
			err := c.Validate()

			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil {
				if tc.ExpectErrors != 0 {
					t2.Error("received no errors but expected errors")
				}
				return
			}
			// - not castable to *Errors, in which case we have the wrong error type ...
			configErrors, ok := err.(util.Errors)
			if !ok {
				t2.Errorf("config.Validate should only return nil or ConfigErrors{...}, got: %v", err)
				return
			}
			// - ConfigErrors, in which case expect a certain number of errors
			errors := configErrors.Errors()
			if len(errors) != tc.ExpectErrors {
				t2.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errors, len(errors))
			}
		})
	}
}
