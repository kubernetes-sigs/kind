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
	"fmt"
	"sigs.k8s.io/kind/pkg/internal/assert"
	"testing"

	"sigs.k8s.io/kind/pkg/errors"
)

func TestClusterValidate(t *testing.T) {
	t.Parallel()
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
			Name: "multiple valid nodes",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Nodes = append(c.Nodes, newDefaultedNode(WorkerRole), newDefaultedNode(WorkerRole))
				return c
			}(),
		},
		{
			Name: "default IPv6",
			Cluster: func() Cluster {
				c := Cluster{}
				c.Networking.IPFamily = IPv6Family
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
			Name: "bogus serviceSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.ServiceSubnet = "aa"
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "bogus apiServerPort",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.APIServerPort = 9999999
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "bogus kubeProxyMode",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.KubeProxyMode = "notiptables"
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "bogus serviceSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.ServiceSubnet = "aa"
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "invalid number of podSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "192.168.0.2/24,2.2.2.0/24"
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "valid dual stack podSubnet and serviceSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "192.168.0.2/24,fd00:1::/25"
				c.Networking.ServiceSubnet = "192.168.0.2/24,fd00:1::/25"
				c.Networking.IPFamily = DualStackFamily
				return c
			}(),
			ExpectErrors: 0,
		},
		{
			Name: "invalid dual stack podSubnet and multiple serviceSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "192.168.0.2/24,fd00:1::/25"
				c.Networking.ServiceSubnet = "192.168.0.2/24,fd00:1::/25,10.0.0.0/16"
				c.Networking.IPFamily = DualStackFamily
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "valid dual stack podSubnet and single stack serviceSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "192.168.0.2/24,fd00:1::/25"
				c.Networking.ServiceSubnet = "192.168.0.2/24"
				c.Networking.IPFamily = DualStackFamily
				return c
			}(),
			ExpectErrors: 0,
		},
		{
			Name: "valid dual stack serviceSubnet and single stack podSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "192.168.0.2/24"
				c.Networking.ServiceSubnet = "192.168.0.2/24,fd00:1::/25"
				c.Networking.IPFamily = DualStackFamily
				return c
			}(),
			ExpectErrors: 0,
		},

		{
			Name: "bad dual stack podSubnet and serviceSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "192.168.0.2/24,2.2.2.0/25"
				c.Networking.ServiceSubnet = "192.168.0.2/24,2.2.2.0/25"
				c.Networking.IPFamily = DualStackFamily
				return c
			}(),
			ExpectErrors: 2,
		},
		{
			Name: "ipv6 family and ipv4 podSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "192.168.0.2/24"
				c.Networking.ServiceSubnet = "192.168.0.2/24"
				c.Networking.IPFamily = IPv6Family
				return c
			}(),
			ExpectErrors: 2,
		},
		{
			Name: "ipv4 family and ipv6 podSubnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = "fd00:1::/25"
				c.Networking.ServiceSubnet = "fd00:1::/25"
				c.Networking.IPFamily = IPv4Family
				return c
			}(),
			ExpectErrors: 2,
		},
		{
			// This test validates the empty podsubnet check. It should never happen
			// in real world since defaulting is happening before the validation step.
			Name: "no pod subnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.PodSubnet = ""
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			// This test validates the empty servicesubnet check. It should never happen
			// in real world since defaulting is happening before the validation step.
			Name: "no service subnet",
			Cluster: func() Cluster {
				c := Cluster{}
				SetDefaultsCluster(&c)
				c.Networking.ServiceSubnet = ""
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
			// get the list of errors
			errs := errors.Errors(err)
			if errs == nil {
				errs = []error{err}
			}
			// we expect a certain number of errors
			if len(errs) != tc.ExpectErrors {
				t.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errs, len(errs))
			}
		})
	}
}

func newDefaultedNode(role NodeRole) Node {
	n := Node{
		Role:  role,
		Image: "myImage:latest",
	}
	SetDefaultsNode(&n)
	return n
}

func TestNodeValidate(t *testing.T) {
	t.Parallel()
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
			TestName: "Unknown role field",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.Role = "ssss"
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Invalid ContainerPort",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.ExtraPortMappings = []PortMapping{
					{
						ContainerPort: 999999999,
						HostPort:      8080,
					},
				}
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Invalid HostPort",
			Node: func() Node {
				cfg := newDefaultedNode(ControlPlaneRole)
				cfg.ExtraPortMappings = []PortMapping{
					{
						ContainerPort: 8080,
						HostPort:      999999999,
					},
				}
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
			// get the list of errors
			errs := errors.Errors(err)
			if errs == nil {
				errs = []error{err}
			}
			// we expect a certain number of errors
			if len(errs) != tc.ExpectErrors {
				t.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errs, len(errs))
			}
		})
	}
}

func TestPortValidate(t *testing.T) {
	cases := []struct {
		TestName    string
		Port        int32
		ExpectError string
	}{
		{
			TestName:    "-1 port",
			Port:        -1,
			ExpectError: "",
		},
		{
			TestName:    "valid port",
			Port:        10,
			ExpectError: "",
		},
		{
			TestName:    "negative port",
			Port:        -2,
			ExpectError: "invalid port number: -2",
		},
		{
			TestName:    "extra port",
			Port:        65536,
			ExpectError: "invalid port number: 65536",
		},
	}

	for _, tc := range cases {
		tc := tc //capture loop variable
		t.Run(tc.TestName, func(t *testing.T) {
			t.Parallel()
			err := validatePort(tc.Port)
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil && len(tc.ExpectError) > 0 {
				t.Errorf("Test failed, unexpected error: %s", tc.ExpectError)
			}

			if err != nil && err.Error() != tc.ExpectError {
				t.Errorf("Test failed, error: %s expected error: %s", err, tc.ExpectError)
			}
		})
	}
}

func TestValidatePortMappings(t *testing.T) {
	newPortMapping := func(addr string, port int, protocol string) PortMapping {
		return PortMapping{
			HostPort:      int32(port),
			ListenAddress: addr,
			Protocol:      PortMappingProtocol(protocol),
		}
	}
	errMsg := "port mapping with same listen address, port and protocol already configured"
	cases := []struct {
		testName     string
		portMappings []PortMapping
		expectErr    string
	}{
		{
			testName: "unique port mappings ipv4",
			portMappings: []PortMapping{
				newPortMapping("127.0.0.1", 80, "UDP"),
				newPortMapping("127.0.0.1", 80, "TCP"),
				newPortMapping("0.0.0.0", 3000, "UDP"),
				newPortMapping("0.0.0.0", 5000, "TCP"),
			},
			expectErr: "",
		},
		{
			testName: "unique port mappings ipv6",
			portMappings: []PortMapping{
				newPortMapping("::1", 80, "UDP"),
				newPortMapping("::1", 80, "TCP"),
				newPortMapping("1e3d:6e85:424d:a011:a72e:9780:5f6f:a6fc", 3000, "UDP"),
				newPortMapping("6516:944d:e070:a1d1:2e91:8437:a6b3:edf9", 5000, "TCP"),
			},
			expectErr: "",
		},
		{
			testName: "exact duplicate port mappings ipv4",
			portMappings: []PortMapping{
				newPortMapping("127.0.0.1", 80, "TCP"),
				newPortMapping("127.0.0.1", 80, "UDP"),
				newPortMapping("127.0.0.1", 80, "TCP"),
			},
			// error expected: exact duplicate
			expectErr: fmt.Sprintf("%s: 127.0.0.1:80/TCP", errMsg),
		},

		{
			testName: "exact duplicate port mappings ipv6",
			portMappings: []PortMapping{
				newPortMapping("::1", 80, "TCP"),
				newPortMapping("::1", 80, "UDP"),
				newPortMapping("::1", 80, "TCP"),
			},
			// error expected: exact duplicate
			expectErr: fmt.Sprintf("%s: [::1]:80/TCP", errMsg),
		},
		{
			testName: "wildcard ipv4 & ipv6",
			portMappings: []PortMapping{
				newPortMapping("127.0.0.1", 80, "TCP"),
				newPortMapping("0.0.0.0", 80, "UDP"),
				newPortMapping("::1", 80, "TCP"),
				newPortMapping("::", 80, "UDP"),
			},
			// error expected: 0.0.0.0 & [::] are same in golang
			// https://github.com/golang/go/issues/48723
			expectErr: fmt.Sprintf("%s: [::]:80/UDP", errMsg),
		},
		{
			testName: "subset already configured ipv4",
			portMappings: []PortMapping{
				newPortMapping("127.0.0.1", 80, "TCP"),
				newPortMapping("0.0.0.0", 80, "TCP"),
			},
			// error expected: subset of 0.0.0.0 -> 127.0.0.1 is already defined for same port and protocol
			expectErr: fmt.Sprintf("%s: 0.0.0.0:80/TCP", errMsg),
		},
		{
			testName: "subset already configured ipv6",
			portMappings: []PortMapping{
				newPortMapping("::1", 80, "TCP"),
				newPortMapping("::", 80, "TCP"),
			},
			// error expected: subset of :: -> ::1 is already defined for same port and protocol
			expectErr: fmt.Sprintf("%s: [::]:80/TCP", errMsg),
		},
		{
			testName: "port mapping already configured via wildcard ipv4",
			portMappings: []PortMapping{
				newPortMapping("0.0.0.0", 80, "TCP"),
				newPortMapping("127.0.0.1", 80, "TCP"),
			},
			// error expected: port mapping is already defined for wildcard interface - 0.0.0.0
			expectErr: fmt.Sprintf("%s: 127.0.0.1:80/TCP", errMsg),
		},
		{
			testName: "port mapping already configured via wildcard ipv6",
			portMappings: []PortMapping{
				newPortMapping("::", 80, "SCTP"),
				newPortMapping("::1", 80, "SCTP"),
			},
			// error expected: port mapping is already defined for wildcard interface - ::
			expectErr: fmt.Sprintf("%s: [::1]:80/SCTP", errMsg),
		},
	}

	for _, tc := range cases {
		tc := tc //capture loop variable
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			err := validatePortMappings(tc.portMappings)
			assert.ExpectError(t, len(tc.expectErr) > 0, err)
		})
	}
}
