// Package swarm is the *stub* multi-host provider for Reduced_kind.
//
// It will eventually satisfy cluster.Provider by sharding `docker run/exec`
// across several hosts that have been pre-joined into a Docker Swarm with a
// shared overlay network ("kind").
//
// Today it only contains skeleton signatures + log lines so we can wire the
// --multihost CLI flag end-to-end before committing to an implementation.
//
// Architecture sketch (to be implemented):
//
//	┌──────────────────────────────────────────────────────────────────┐
//	│   Reduced_kind --multihost create                                │
//	│   ─────────────────────────────                                  │
//	│   1. EnsureSwarm   on the manager host                           │
//	│      → docker swarm init --advertise-addr <managerIP>            │
//	│      → returns BuildJoinInstruction(...)                         │
//	│   2. RunJoinOnWorker  on each worker host                        │
//	│      → docker swarm join --token <T> <managerIP>:2377            │
//	│   3. EnsureOverlay  on the manager host                          │
//	│      → docker network create -d overlay --attachable kind        │
//	│   4. provider.Provision(cfg)                                     │
//	│      → for each cfg.Nodes[i]:                                    │
//	│          docker --context=<Node.Host> run --network=kind ...     │
//	└──────────────────────────────────────────────────────────────────┘
package swarm

import (
	"fmt"

	"reducedkind/cluster"
	"reducedkind/config"
)

// JoinSpec is everything a worker host needs to join the swarm.
type JoinSpec struct {
	ManagerAddr string // "172.16.193.6:2377"
	Token       string // SWMTKN-1-... (issued by `docker swarm init`)
	Network     string // overlay network name, default "kind"
}

// New returns the stub multi-host provider.
//
// TODO: real implementation.  Should accept a list of hosts (docker
// contexts), validate that they're already joined into the same Swarm,
// and remember the mapping so Provision can place each node on the right
// host.
func New() cluster.Provider {
	fmt.Println("[multihost] swarm.New() — stub provider, no real Docker calls yet")
	return &Provider{}
}

// EnsureSwarm initialises a Swarm on the manager host and returns the JoinSpec
// that workers need.
//
// TODO: real implementation will run, on the manager:
//
//	docker swarm init --advertise-addr <managerIP>
//
// then parse the join command from the output and fill JoinSpec.
func EnsureSwarm(managerIP string) (JoinSpec, error) {
	fmt.Printf("[multihost] EnsureSwarm(managerIP=%q) — stub\n", managerIP)
	return JoinSpec{
		ManagerAddr: fmt.Sprintf("%s:2377", managerIP),
		Token:       "SWMTKN-STUB-TOKEN",
		Network:     "kind",
	}, nil
}

// BuildJoinInstruction returns the shell command another host should run to
// join the swarm as a worker.  Useful both for displaying to the user
// ("paste this on host B") and for programmatic execution over SSH.
//
// TODO: real implementation will simply format the JoinSpec.
func BuildJoinInstruction(spec JoinSpec) string {
	cmd := fmt.Sprintf("docker swarm join --token %s %s",
		spec.Token, spec.ManagerAddr)
	fmt.Printf("[multihost] BuildJoinInstruction → %s\n", cmd)
	return cmd
}

// RunJoinOnWorker SSH-es into the worker host and runs the join command.
//
// TODO: real implementation will use ssh / docker context to execute the
// join.  Right now it just prints what it *would* do.
func RunJoinOnWorker(workerHost string, spec JoinSpec) error {
	fmt.Printf("[multihost] RunJoinOnWorker(host=%q) would run: %s\n",
		workerHost, BuildJoinInstruction(spec))
	return nil
}

// EnsureOverlay creates the cross-host overlay network that node containers
// will attach to.  All nodes (regardless of host) share this one network.
//
// TODO: real implementation will run, on the manager:
//
//	docker network create -d overlay --attachable kind
func EnsureOverlay(name string) error {
	fmt.Printf("[multihost] EnsureOverlay(name=%q) — stub (would: docker network create -d overlay --attachable %s)\n", name, name)
	return nil
}

// Provider is the stub cluster.Provider for Swarm.  All methods log the call
// and return "not implemented".  Replacing them one by one is the next
// implementation milestone.
type Provider struct{}

func (p *Provider) Provision(cfg *config.Cluster) error {
	fmt.Printf("[multihost] Provider.Provision(cluster=%q, nodes=%d) — stub\n",
		cfg.Name, len(cfg.Nodes))
	for i, n := range cfg.Nodes {
		host := n.Host
		if host == "" {
			host = "(default host)"
		}
		fmt.Printf("[multihost]   node %d: role=%s host=%s image=%s\n",
			i, n.Role, host, n.Image)
	}
	return fmt.Errorf("multihost provider not implemented yet")
}

func (p *Provider) ListNodes(clusterName string) ([]cluster.Node, error) {
	fmt.Printf("[multihost] Provider.ListNodes(%q) — stub\n", clusterName)
	return nil, nil // empty list lets `delete` no-op cleanly
}

func (p *Provider) DeleteNodes(nodes []cluster.Node) error {
	fmt.Printf("[multihost] Provider.DeleteNodes(%d nodes) — stub\n", len(nodes))
	return nil
}

func (p *Provider) GetAPIServerEndpoint(clusterName string) (string, error) {
	fmt.Printf("[multihost] Provider.GetAPIServerEndpoint(%q) — stub\n", clusterName)
	return "", fmt.Errorf("not implemented")
}
