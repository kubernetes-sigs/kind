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

	utilpointer "k8s.io/utils/pointer"
)

func TestAdd(t *testing.T) {
	cases := []struct {
		TestName                     string
		Nodes                        []*Node
		ExpectNodes                  []string
		ExpectControlPlanes          []string
		ExpectBootStrapControlPlane  *string
		ExpectSecondaryControlPlanes []string
		ExpectWorkers                []string
		ExpectEtcd                   *string
		ExpectLoadBalancer           *string
		ExpectError                  bool
	}{
		{
			TestName:                     "Defaults/Empty config should give empty Nodes",
			ExpectNodes:                  nil,
			ExpectControlPlanes:          nil,
			ExpectBootStrapControlPlane:  nil,
			ExpectSecondaryControlPlanes: nil,
			ExpectWorkers:                nil,
			ExpectEtcd:                   nil,
			ExpectLoadBalancer:           nil,
			ExpectError:                  false,
		},
		{
			TestName: "Single control plane get properly assigned to bootstrap control-plane",
			Nodes: []*Node{
				{Role: ControlPlaneRole},
			},
			ExpectNodes:                  []string{"control-plane"},
			ExpectControlPlanes:          []string{"control-plane"},
			ExpectBootStrapControlPlane:  utilpointer.StringPtr("control-plane"),
			ExpectSecondaryControlPlanes: nil,
			ExpectError:                  false,
		},
		{
			TestName: "Control planes get properly splitted between bootstrap and secondary control-planes",
			Nodes: []*Node{
				{Role: ControlPlaneRole},
				{Role: ControlPlaneRole},
				{Role: ControlPlaneRole},
			},
			ExpectNodes:                  []string{"control-plane1", "control-plane2", "control-plane3"},
			ExpectControlPlanes:          []string{"control-plane1", "control-plane2", "control-plane3"},
			ExpectBootStrapControlPlane:  utilpointer.StringPtr("control-plane1"),
			ExpectSecondaryControlPlanes: []string{"control-plane2", "control-plane3"},
			ExpectError:                  false,
		},
		{
			TestName: "Single control plane get properly named if more than one node exists",
			Nodes: []*Node{
				{Role: ControlPlaneRole},
				{Role: WorkerRole},
			},
			ExpectNodes:                  []string{"control-plane", "worker"},
			ExpectControlPlanes:          []string{"control-plane"},
			ExpectBootStrapControlPlane:  utilpointer.StringPtr("control-plane"),
			ExpectSecondaryControlPlanes: nil,
			ExpectWorkers:                []string{"worker"},
			ExpectError:                  false,
		},
		{
			TestName: "Full HA cluster", // NB. This test case test that provisioning order is applied to all the node lists as well
			Nodes: []*Node{
				{Role: WorkerRole},
				{Role: ControlPlaneRole},
				{Role: ExternalEtcdRole},
				{Role: ControlPlaneRole},
				{Role: WorkerRole},
				{Role: ControlPlaneRole},
				{Role: ExternalLoadBalancerRole},
			},
			ExpectNodes:                  []string{"etcd", "lb", "control-plane1", "control-plane2", "control-plane3", "worker1", "worker2"},
			ExpectControlPlanes:          []string{"control-plane1", "control-plane2", "control-plane3"},
			ExpectBootStrapControlPlane:  utilpointer.StringPtr("control-plane1"),
			ExpectSecondaryControlPlanes: []string{"control-plane2", "control-plane3"},
			ExpectWorkers:                []string{"worker1", "worker2"},
			ExpectEtcd:                   utilpointer.StringPtr("etcd"),
			ExpectLoadBalancer:           utilpointer.StringPtr("lb"),
			ExpectError:                  false,
		},
		{
			TestName: "Fails because two etcds Nodes are added",
			Nodes: []*Node{
				{Role: ExternalEtcdRole},
				{Role: ExternalEtcdRole},
			},
			ExpectError: true,
		},
		{
			TestName: "Fails because two load balancer Nodes are added",
			Nodes: []*Node{
				{Role: ExternalLoadBalancerRole},
				{Role: ExternalLoadBalancerRole},
			},
			ExpectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t2 *testing.T) {
			// Adding Nodes to the config until first error or completing all Nodes
			var cfg = Config{}
			var err error
			for _, n := range c.Nodes {
				if e := cfg.Add(n); e != nil {
					err = e
					break
				}
			}
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t2.Fatalf("unexpected error while adding Nodes: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t2.Fatalf("unexpected lack or error while adding Nodes")
				}
			}

			// Fail if Nodes does not match
			checkNodeList(t2, cfg.Nodes(), c.ExpectNodes)

			// Fail if fields derived from Nodes does not match
			checkNodeList(t2, cfg.ControlPlanes(), c.ExpectControlPlanes)
			checkNode(t2, cfg.BootStrapControlPlane(), c.ExpectBootStrapControlPlane)
			checkNodeList(t2, cfg.SecondaryControlPlanes(), c.ExpectSecondaryControlPlanes)
			checkNodeList(t2, cfg.Workers(), c.ExpectWorkers)
			checkNode(t2, cfg.ExternalEtcd(), c.ExpectEtcd)
			checkNode(t2, cfg.ExternalLoadBalancer(), c.ExpectLoadBalancer)
		})
	}
}

func checkNode(t *testing.T, n *Node, name *string) {
	if (n == nil) != (name == nil) {
		t.Errorf("expected %v node, saw %v", name, n)
	}

	if n == nil {
		return
	}

	if n.Name != *name {
		t.Errorf("expected %v node, saw %v", name, n.Name)
	}
}

func checkNodeList(t *testing.T, list NodeList, names []string) {
	if len(list) != len(names) {
		t.Errorf("expected %d nodes, saw %d", len(names), len(list))
		return
	}

	for i, name := range names {
		if list[i].Name != name {
			t.Errorf("expected %q node at position %d, saw %q", name, i, list[i].Name)
		}
	}
}
