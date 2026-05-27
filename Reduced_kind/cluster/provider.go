// Package cluster defines the runtime-agnostic interfaces a Reduced_kind
// provider must implement to bring up a Kubernetes cluster.
//
// The split between Provider and Node mirrors kind:
//
//   - Provider is "cluster-level": create/list/delete the set of node containers,
//     and answer questions about cluster-wide state (API server endpoint).
//   - Node is "node-level": run a command inside one node container,
//     write a file into it, look up its IP.
//
// Single-host and multi-host implementations both satisfy these interfaces.
// The orchestration code in create.go is identical for both.
package cluster

import "reducedkind/config"

// Provider creates and manipulates the container substrate that nodes run on.
//
// The single-host implementation in package docker uses `docker run/exec/ps`
// against the local daemon.  A multi-host implementation can satisfy the same
// interface by sharding `docker --context=<host>` calls across hosts.
type Provider interface {
	// Provision creates one container per cfg.Nodes and ensures the network
	// they live on exists.  Containers come up but Kubernetes is not yet
	// initialised on them.
	Provision(cfg *config.Cluster) error

	// ListNodes returns the nodes (running or stopped) belonging to the
	// named cluster, in cfg-declared order.
	ListNodes(clusterName string) ([]Node, error)

	// DeleteNodes tears down the supplied nodes and any provider-managed
	// resources (volumes, networks if last cluster).
	DeleteNodes(nodes []Node) error

	// GetAPIServerEndpoint returns the externally-reachable host:port for
	// the cluster's API server.  Used to rewrite kubeconfig.
	GetAPIServerEndpoint(clusterName string) (string, error)
}

// Node is a handle to one node container.  It hides the difference between
// "exec on local docker" and "exec on remote docker over a context".
type Node interface {
	// Name is the container name (e.g. "kind-control-plane").
	Name() string

	// Role returns "control-plane" or "worker".
	Role() string

	// IP returns the node's address inside the cluster network.
	IP() (string, error)

	// Exec runs a command inside the node container and returns its
	// combined output.
	Exec(cmd string, args ...string) ([]byte, error)

	// WriteFile creates path inside the node with the supplied contents.
	WriteFile(path, content string) error
}
