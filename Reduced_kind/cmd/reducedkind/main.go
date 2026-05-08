// reducedkind is a minimal kind clone.
//
// Modes:
//
//	reducedkind create [name]
//	    single-host: spawns node containers on the local Docker daemon
//
//	reducedkind --multihost --hosts <ctx>=<addr>[,...] create [name]
//	    multi-host: spawns nodes across the listed Docker contexts,
//	    attached to a shared Swarm overlay network "kind".  The first
//	    host is treated as the swarm manager.
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
			"Example: --hosts default=172.16.193.6,ecotype-35=172.16.193.35")
	bootstrap := flag.Bool("bootstrap-swarm", false,
		"multihost: run 'docker swarm init' on the manager and 'swarm join' on each worker before creating the cluster")
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

	provider, err := pickProvider(*multihost, *hostsFlag, *bootstrap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	switch verb {
	case "create":
		cfg := &config.Cluster{Name: name}
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

// pickProvider chooses between single-host and multi-host providers.
// In multi-host mode, parses --hosts and optionally bootstraps the
// swarm before returning the provider.
func pickProvider(multihost bool, hostsFlag string, bootstrap bool) (cluster.Provider, error) {
	if !multihost {
		fmt.Println("==> mode: SINGLE-HOST (local Docker daemon)")
		return docker.New(), nil
	}

	fmt.Println("==> mode: MULTI-HOST (Docker Swarm overlay)")
	hosts, err := parseHosts(hostsFlag)
	if err != nil {
		return nil, err
	}
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
