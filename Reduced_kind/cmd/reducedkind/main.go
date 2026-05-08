// reducedkind is a minimal kind clone.
//
// Single-host today (default).  With --multihost, dispatches to a stub
// Swarm-based provider in package swarm/ — that branch isn't implemented
// yet, but the CLI plumbing is in place.
//
// Usage:
//
//	reducedkind [--multihost] create [name]
//	reducedkind [--multihost] delete [name]
package main

import (
	"flag"
	"fmt"
	"os"

	"reducedkind/actions"
	"reducedkind/cluster"
	"reducedkind/config"
	"reducedkind/docker"
	"reducedkind/swarm"
)

func main() {
	multihost := flag.Bool("multihost", false,
		"create the cluster across multiple hosts via Docker Swarm (stub)")
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

	provider := pickProvider(*multihost)

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

// pickProvider chooses between the local-Docker provider (default) and the
// Swarm-based multi-host provider.  Prints the chosen mode so the user
// always knows which branch they're in.
func pickProvider(multihost bool) cluster.Provider {
	if multihost {
		fmt.Println("==> mode: MULTI-HOST (Docker Swarm overlay)")
		fmt.Println("    NOTE: multihost provider is a stub; this run will not actually start a cluster.")
		demoSwarmSetup()
		return swarm.New()
	}
	fmt.Println("==> mode: SINGLE-HOST (local Docker daemon)")
	return docker.New()
}

// demoSwarmSetup walks through the swarm-bootstrap steps the (future)
// multi-host implementation will perform, using the stub helpers.  Today
// it just prints what each step *would* do; replace with real calls when
// implementing.
func demoSwarmSetup() {
	const (
		demoManagerIP = "172.16.193.6"
		demoWorker    = "ecotype-35"
	)
	fmt.Println("--- swarm bootstrap (stub) ---")
	spec, err := swarm.EnsureSwarm(demoManagerIP)
	if err != nil {
		fmt.Println("EnsureSwarm:", err)
		return
	}
	cmd := swarm.BuildJoinInstruction(spec)
	fmt.Println("    → join command for workers:", cmd)
	if err := swarm.RunJoinOnWorker(demoWorker, spec); err != nil {
		fmt.Println("RunJoinOnWorker:", err)
		return
	}
	if err := swarm.EnsureOverlay(spec.Network); err != nil {
		fmt.Println("EnsureOverlay:", err)
		return
	}
	fmt.Println("--- swarm bootstrap done ---")
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: reducedkind [--multihost] {create|delete} [name]")
	flag.PrintDefaults()
}
