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

	"sigs.k8s.io/kind/pkg/errors"
)

func TestClusterValidate(t *testing.T) {
	cases := []struct {
		Name         string
		Cluster      Cluster
		ExpectErrors int
	}{
		{
			Name: "Defaulted",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				return c
			}(),
		},
		{
			Name: "bogus podSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "aa"
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "missing control-plane",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Nodes = []Node{}
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "bogus node",
			Cluster: func() Cluster {
				c := Cluster{}
				n, n2 := Node{}, Node{}
				n.Role = "bogus"
				c.Nodes = []Node{n, n2}
				SetDefaultsCluster(&c)
				return c
			}(),
			ExpectErrors: 1,
		},
	}

	for _, tc := range cases {
		tc := tc //capture loop variable
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			err := tc.Cluster.Validate()
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil {
				if tc.ExpectErrors != 0 {
					t.Error("received no errors but expected errors for case")
				}
				return
			}
			// - doesn't container errors, in which case we have the wrong error type ...
			errs := errors.Errors(err)
			if errs == nil {
				t.Errorf("config.Validate should only return nil or ConfigErrors{...}, got: %v", err)
				return
			}
			// - errors, in which case expect a certain number of errors
			if len(errs) != tc.ExpectErrors {
				t.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errs, len(errs))
			}
		})
	}
}

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
			TestName:     "Canonical node 2",
			Node:         newDefaultedNode(WorkerRole),
			ExpectErrors: 0,
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
		tc := tc //capture loop variable
		t.Run(tc.TestName, func(t *testing.T) {
			t.Parallel()
			err := tc.Node.Validate()
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil {
				if tc.ExpectErrors != 0 {
					t.Error("received no errors but expected errors for case")
				}
				return
			}
			// - doesn't container errors, in which case we have the wrong error type ...
			errs := errors.Errors(err)
			if errs == nil {
				t.Errorf("config.Validate should only return nil or ConfigErrors{...}, got: %v", err)
				return
			}
			// - errors, in which case expect a certain number of errors
			if len(errs) != tc.ExpectErrors {
				t.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errs, len(errs))
			}
		})
	}
}
