// reducedkind is a minimal kind clone.  Single-host today, designed to be
// extended to multi-host (Docker Swarm) later.  See ../../DESIGN.md.
//
// Usage:
//
//	reducedkind create [name]
//	reducedkind delete [name]
package main

import (
	"fmt"
	"os"

	"reducedkind/actions"
	"reducedkind/cluster"
	"reducedkind/config"
	"reducedkind/docker"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	name := config.DefaultClusterName
	if len(os.Args) >= 3 {
		name = os.Args[2]
	}

	provider := docker.New()

	switch os.Args[1] {
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

func usage() {
	fmt.Fprintln(os.Stderr, "usage: reducedkind {create|delete} [name]")
}
