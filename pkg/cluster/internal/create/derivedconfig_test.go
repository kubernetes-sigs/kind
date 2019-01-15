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

package create

import (
	"testing"

	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/kind/pkg/cluster/config"
)

func TestDeriveInfo(t *testing.T) {
	cases := []struct {
		TestName                     string
		Nodes                        []config.Node
		ExpectReplicas               []string
		ExpectControlPlanes          []string
		ExpectBootStrapControlPlane  *string
		ExpectSecondaryControlPlanes []string
		ExpectWorkers                []string
		ExpectEtcd                   *string
		ExpectLoadBalancer           *string
		ExpectError                  bool
	}{
		{
			TestName:                     "Defaults/Empty config should give empty replicas",
			ExpectReplicas:               nil,
			ExpectControlPlanes:          nil,
			ExpectBootStrapControlPlane:  nil,
			ExpectSecondaryControlPlanes: nil,
			ExpectWorkers:                nil,
			ExpectEtcd:                   nil,
			ExpectLoadBalancer:           nil,
			ExpectError:                  false,
		},
		{
			TestName: "Single control plane Node get properly assigned to bootstrap control-plane",
			Nodes: []config.Node{
				{Role: config.ControlPlaneRole},
			},
			ExpectReplicas:               []string{"control-plane"},
			ExpectControlPlanes:          []string{"control-plane"},
			ExpectBootStrapControlPlane:  utilpointer.StringPtr("control-plane"),
			ExpectSecondaryControlPlanes: nil,
			ExpectError:                  false,
		},
		{
			TestName: "Control planes Nodes get properly splitted between bootstrap and secondary control-planes",
			Nodes: []config.Node{
				{Role: config.ControlPlaneRole, Replicas: utilpointer.Int32Ptr(3)},
			},
			ExpectReplicas:               []string{"control-plane1", "control-plane2", "control-plane3"},
			ExpectControlPlanes:          []string{"control-plane1", "control-plane2", "control-plane3"},
			ExpectBootStrapControlPlane:  utilpointer.StringPtr("control-plane1"),
			ExpectSecondaryControlPlanes: []string{"control-plane2", "control-plane3"},
			ExpectError:                  false,
		},
		{
			TestName: "Full HA cluster", // NB. This test case test that provisioning order is applied to all the node lists as well
			Nodes: []config.Node{
				{Role: config.WorkerRole},
				{Role: config.ControlPlaneRole},
				{Role: config.ExternalEtcdRole},
				{Role: config.ControlPlaneRole},
				{Role: config.WorkerRole},
				{Role: config.ControlPlaneRole},
				{Role: config.ExternalLoadBalancerRole},
			},
			ExpectReplicas:               []string{"etcd", "lb", "control-plane1", "control-plane2", "control-plane3", "worker1", "worker2"},
			ExpectControlPlanes:          []string{"control-plane1", "control-plane2", "control-plane3"},
			ExpectBootStrapControlPlane:  utilpointer.StringPtr("control-plane1"),
			ExpectSecondaryControlPlanes: []string{"control-plane2", "control-plane3"},
			ExpectWorkers:                []string{"worker1", "worker2"},
			ExpectEtcd:                   utilpointer.StringPtr("etcd"),
			ExpectLoadBalancer:           utilpointer.StringPtr("lb"),
			ExpectError:                  false,
		},
		{
			TestName: "Full HA cluster with replicas",
			Nodes: []config.Node{
				{Role: config.WorkerRole, Replicas: utilpointer.Int32Ptr(2)},
				{Role: config.ControlPlaneRole, Replicas: utilpointer.Int32Ptr(3)},
				{Role: config.ExternalEtcdRole},
				{Role: config.ExternalLoadBalancerRole},
			},
			ExpectReplicas:               []string{"etcd", "lb", "control-plane1", "control-plane2", "control-plane3", "worker1", "worker2"},
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
			Nodes: []config.Node{
				{Role: config.ExternalEtcdRole},
				{Role: config.ExternalEtcdRole},
			},
			ExpectError: true,
		},
		{
			TestName: "Fails because two load balancer Nodes are added",
			Nodes: []config.Node{
				{Role: config.ExternalLoadBalancerRole},
				{Role: config.ExternalLoadBalancerRole},
			},
			ExpectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			// Adding Nodes to the config and deriving infos
			var cfg = &config.Config{Nodes: c.Nodes}
			derived, err := Derive(cfg)
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t.Fatalf("unexpected error while Deriving infos: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t.Fatalf("unexpected lack or error while Deriving infos")
				}
			}

			// Fail if Nodes does not match
			checkReplicaList(t, derived.AllReplicas(), c.ExpectReplicas)

			// Fail if fields derived from Nodes does not match
			checkReplicaList(t, derived.ControlPlanes(), c.ExpectControlPlanes)
			checkNode(t, derived.BootStrapControlPlane(), c.ExpectBootStrapControlPlane)
			checkReplicaList(t, derived.SecondaryControlPlanes(), c.ExpectSecondaryControlPlanes)
			checkReplicaList(t, derived.Workers(), c.ExpectWorkers)
			checkNode(t, derived.ExternalEtcd(), c.ExpectEtcd)
			checkNode(t, derived.ExternalLoadBalancer(), c.ExpectLoadBalancer)
		})
	}
}

func checkNode(t *testing.T, n *NodeReplica, name *string) {
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

func checkReplicaList(t *testing.T, list ReplicaList, names []string) {
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
