// reducedkind is a minimal kind clone.
//
// Modes:
//
//	reducedkind create [name]
//	    single-host: spawns one control-plane container on the local
//	    Docker daemon.
//
//	reducedkind --nodes N create [name]
//	    single-host, multi-node: 1 control-plane + (N-1) workers, all
//	    on the local Docker daemon (kind's classic "config with N nodes"
//	    layout).
//
//	reducedkind --multihost --hosts <ctx>=<addr>[,...] create [name]
//	    multi-host, one node per host: 1 control-plane on the first
//	    host, 1 worker on each remaining host, on a Swarm overlay.
//
//	reducedkind --multihost --hosts ... --nodes N create [name]
//	    multi-host, N total nodes round-robined across hosts.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"reducedkind/actions"
	"reducedkind/cluster"
	"reducedkind/config"
	"reducedkind/docker"
	"reducedkind/swarm"
)

func main() {
	multihost := flag.Bool("multihost", false,
		"create the cluster across multiple hosts via Docker Swarm")
	hostsFlag := flag.String("hosts", "",
		"multihost: comma-separated <docker-context>=<external-addr> pairs (first = manager). "+
			"Example: --hosts default=172.16.193.6,ecotype-48=172.16.193.48")
	bootstrap := flag.Bool("bootstrap-swarm", false,
		"multihost: run 'docker swarm init' on the manager and 'swarm join' on each worker before creating the cluster")
	nodes := flag.Int("nodes", 0,
		"total number of K8s nodes (1 control-plane + (N-1) workers). "+
			"Default 0 means: single-host=1 node, multi-host=one per --hosts entry")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}
	verb := args[0]
	name := config.DefaultClusterName
	if len(args) >= 2 {
		name = args[1]
	}

	// Parse the hosts list once, up front; both pickProvider and
	// buildCluster need it.
	var hosts []swarm.Host
	if *multihost {
		var err error
		hosts, err = parseHosts(*hostsFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	}

	provider, err := pickProvider(*multihost, hosts, *bootstrap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	switch verb {
	case "create":
		cfg := buildCluster(name, hosts, *nodes)
		if err := cluster.Create(provider, cfg, actions.All()); err != nil {
			fmt.Fprintln(os.Stderr, "create failed:", err)
			os.Exit(1)
		}
		ep, _ := provider.GetAPIServerEndpoint(name)
		fmt.Printf("cluster %q ready, API server at %s\n", name, ep)

	case "delete":
		if err := cluster.Delete(provider, name); err != nil {
			fmt.Fprintln(os.Stderr, "delete failed:", err)
			os.Exit(1)
		}
		fmt.Printf("cluster %q deleted\n", name)

	default:
		usage()
		os.Exit(2)
	}
}

// buildCluster decides the cluster topology from the parsed flags.
//
// The first node is always the control-plane; the rest are workers.
// Naming inside the providers takes care of unique container names
// (foo-control-plane, foo-worker, foo-worker2, ...).
//
// Decision table:
//
//	hosts=[]   nodes<=0  → 1 CP, local docker (kind default)
//	hosts=[]   nodes=N   → 1 CP + (N-1) workers, all on local docker
//	hosts=[H]  nodes<=0  → 1 CP on H
//	hosts=[H]  nodes=N   → 1 CP + (N-1) workers, all on H
//	hosts=H..  nodes<=0  → 1 CP on H[0] + 1 worker per remaining host
//	hosts=H..  nodes=N   → N nodes round-robined over hosts
//	                       (host[0] gets the CP, others cycle)
func buildCluster(name string, hosts []swarm.Host, nodes int) *config.Cluster {
	cfg := &config.Cluster{Name: name}

	// Single-host (no --multihost).
	if len(hosts) == 0 {
		if nodes <= 1 {
			return cfg // SetDefaults will add a single control-plane
		}
		cfg.Nodes = make([]config.Node, 0, nodes)
		cfg.Nodes = append(cfg.Nodes, config.Node{Role: config.ControlPlaneRole})
		for i := 1; i < nodes; i++ {
			cfg.Nodes = append(cfg.Nodes, config.Node{Role: config.WorkerRole})
		}
		return cfg
	}

	// Multi-host with --nodes unspecified: legacy "1 per host" layout.
	if nodes <= 0 {
		cfg.Nodes = []config.Node{{Role: config.ControlPlaneRole}}
		for _, h := range hosts[1:] {
			cfg.Nodes = append(cfg.Nodes, config.Node{
				Role: config.WorkerRole,
				Host: h.Context,
			})
		}
		return cfg
	}

	// Multi-host with --nodes N: round-robin across hosts.
	cfg.Nodes = make([]config.Node, 0, nodes)
	for i := 0; i < nodes; i++ {
		role := config.WorkerRole
		if i == 0 {
			role = config.ControlPlaneRole
		}
		cfg.Nodes = append(cfg.Nodes, config.Node{
			Role: role,
			Host: hosts[i%len(hosts)].Context,
		})
	}
	return cfg
}

// pickProvider chooses between single-host and multi-host providers.
// In multi-host mode it optionally bootstraps the swarm.
func pickProvider(multihost bool, hosts []swarm.Host, bootstrap bool) (cluster.Provider, error) {
	if !multihost {
		fmt.Println("==> mode: SINGLE-HOST (local Docker daemon)")
		return docker.New(), nil
	}

	fmt.Println("==> mode: MULTI-HOST (Docker Swarm overlay)")
	fmt.Printf("    manager: %s (%s)\n", hosts[0].Context, hosts[0].Addr)
	for _, w := range hosts[1:] {
		fmt.Printf("    worker:  %s (%s)\n", w.Context, w.Addr)
	}

	if bootstrap {
		fmt.Println("--- bootstrapping swarm ---")
		spec, err := swarm.Bootstrap(hosts[0], hosts[1:], "kind")
		if err != nil {
			return nil, fmt.Errorf("swarm bootstrap: %w", err)
		}
		fmt.Println("    join command:", swarm.BuildJoinInstruction(spec))
	}

	return swarm.New(hosts, "kind"), nil
}

// parseHosts turns "ctx1=addr1,ctx2=addr2" into []swarm.Host.
func parseHosts(s string) ([]swarm.Host, error) {
	if s == "" {
		return nil, fmt.Errorf("--multihost requires --hosts <ctx>=<addr>[,<ctx>=<addr>...]")
	}
	var out []swarm.Host
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 || eq == len(pair)-1 {
			return nil, fmt.Errorf("invalid host spec %q (want <ctx>=<addr>)", pair)
		}
		out = append(out, swarm.Host{
			Context: strings.TrimSpace(pair[:eq]),
			Addr:    strings.TrimSpace(pair[eq+1:]),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("--hosts produced an empty list")
	}
	return out, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: reducedkind [--multihost --hosts ...] [--bootstrap-swarm] {create|delete} [name]")
	flag.PrintDefaults()
}
