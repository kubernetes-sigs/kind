package main

import (
	"testing"

	"reducedkind/config"
	"reducedkind/swarm"
)

// TestBuildCluster covers every (hosts × nodes) combination in the
// decision table at the top of buildCluster.
func TestBuildCluster(t *testing.T) {
	h := func(ctx string) swarm.Host { return swarm.Host{Context: ctx, Addr: "1.2.3.4"} }

	cases := []struct {
		name      string
		hosts     []swarm.Host
		nodes     int
		wantRoles []config.NodeRole // empty means "expect empty cfg.Nodes"
		wantHosts []string          // "" = local / not pinned
	}{
		{
			name:      "single-host default",
			hosts:     nil,
			nodes:     0,
			wantRoles: nil, // empty; SetDefaults will add 1 CP later
		},
		{
			name:      "single-host --nodes 1",
			hosts:     nil,
			nodes:     1,
			wantRoles: nil,
		},
		{
			name:      "single-host --nodes 3",
			hosts:     nil,
			nodes:     3,
			wantRoles: []config.NodeRole{config.ControlPlaneRole, config.WorkerRole, config.WorkerRole},
			wantHosts: []string{"", "", ""},
		},
		{
			name:      "multihost 2 hosts default",
			hosts:     []swarm.Host{h("h1"), h("h2")},
			nodes:     0,
			wantRoles: []config.NodeRole{config.ControlPlaneRole, config.WorkerRole},
			wantHosts: []string{"", "h2"},
		},
		{
			name:      "multihost 3 hosts default",
			hosts:     []swarm.Host{h("h1"), h("h2"), h("h3")},
			nodes:     0,
			wantRoles: []config.NodeRole{config.ControlPlaneRole, config.WorkerRole, config.WorkerRole},
			wantHosts: []string{"", "h2", "h3"},
		},
		{
			name:      "multihost 2 hosts --nodes 5 (round-robin)",
			hosts:     []swarm.Host{h("h1"), h("h2")},
			nodes:     5,
			wantRoles: []config.NodeRole{config.ControlPlaneRole, config.WorkerRole, config.WorkerRole, config.WorkerRole, config.WorkerRole},
			wantHosts: []string{"h1", "h2", "h1", "h2", "h1"},
		},
		{
			name:      "multihost 1 host --nodes 4",
			hosts:     []swarm.Host{h("h1")},
			nodes:     4,
			wantRoles: []config.NodeRole{config.ControlPlaneRole, config.WorkerRole, config.WorkerRole, config.WorkerRole},
			wantHosts: []string{"h1", "h1", "h1", "h1"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := buildCluster("demo", c.hosts, c.nodes)
			if len(cfg.Nodes) != len(c.wantRoles) {
				t.Fatalf("len(Nodes)=%d, want %d (nodes=%v)",
					len(cfg.Nodes), len(c.wantRoles), cfg.Nodes)
			}
			for i, want := range c.wantRoles {
				if cfg.Nodes[i].Role != want {
					t.Errorf("node[%d].Role=%q, want %q", i, cfg.Nodes[i].Role, want)
				}
				if i < len(c.wantHosts) && cfg.Nodes[i].Host != c.wantHosts[i] {
					t.Errorf("node[%d].Host=%q, want %q", i, cfg.Nodes[i].Host, c.wantHosts[i])
				}
			}
		})
	}
}
