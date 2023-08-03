/*
Copyright 2021 The Kubernetes Authors.

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

	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestClusterHasIPv6(t *testing.T) {
	cases := []struct {
		Name     string
		c        *Cluster
		expected bool
	}{
		{
			Name: "IPv6",
			c: &Cluster{
				Networking: Networking{
					IPFamily: IPv6Family,
				},
			},
			expected: true,
		},
		{
			Name: "IPv4",
			c: &Cluster{
				Networking: Networking{
					IPFamily: IPv4Family,
				},
			},
			expected: false,
		},
		{
			Name: "DualStack",
			c: &Cluster{
				Networking: Networking{
					IPFamily: DualStackFamily,
				},
			},
			expected: true,
		},
	}
	for _, tc := range cases {
		tc := tc // capture loop var
		t.Run(tc.Name, func(t *testing.T) {
			r := ClusterHasIPv6(tc.c)
			assert.BoolEqual(t, tc.expected, r)
		})
	}
}

func TestClusterHasImplicitLoadBalancer(t *testing.T) {
	cases := []struct {
		Name     string
		c        *Cluster
		expected bool
	}{
		{
			Name: "One Node",
			c: &Cluster{
				Nodes: []Node{
					{Role: ControlPlaneRole},
				},
			},
			expected: false,
		},
		{
			Name: "Two Control Planes",
			c: &Cluster{
				Nodes: []Node{
					{Role: ControlPlaneRole},
					{Role: ControlPlaneRole},
				},
			},
			expected: true,
		},
		{
			Name: "Three Control Planes",
			c: &Cluster{
				Nodes: []Node{
					{Role: ControlPlaneRole},
					{Role: ControlPlaneRole},
					{Role: ControlPlaneRole},
				},
			},
			expected: true,
		},
		{
			Name: "One Control Plane, Multiple Workers",
			c: &Cluster{
				Nodes: []Node{
					{Role: ControlPlaneRole},
					{Role: WorkerRole},
					{Role: WorkerRole},
					{Role: WorkerRole},
				},
			},
			expected: false,
		},
		{
			Name: "Multiple Control Planes, Multiple Workers",
			c: &Cluster{
				Nodes: []Node{
					{Role: ControlPlaneRole},
					{Role: ControlPlaneRole},
					{Role: ControlPlaneRole},
					{Role: WorkerRole},
					{Role: WorkerRole},
					{Role: WorkerRole},
				},
			},
			expected: true,
		},
	}
	for _, tc := range cases {
		tc := tc // capture loop var
		t.Run(tc.Name, func(t *testing.T) {
			r := ClusterHasImplicitLoadBalancer(tc.c)
			assert.BoolEqual(t, tc.expected, r)
		})
	}
}
