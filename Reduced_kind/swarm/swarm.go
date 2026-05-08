// Package swarm is the multi-host Reduced_kind provider built on top of a
// pre-existing Docker Swarm.
//
// Assumptions about the user's environment
// ─────────────────────────────────────────
//   - Each host already runs a Docker daemon.
//   - One host is a Swarm manager, others are workers.  This package can
//     bootstrap that for you (EnsureSwarm + RunJoinOnWorker), but most
//     production users will have done it manually.
//   - The user has created `docker context` entries pointing at every
//     host that should run a node container.  Example:
//         docker context create ecotype-35 --docker host=ssh://root@ecotype-35.nantes.grid5000.fr
//     Then we issue `docker --context=ecotype-35 ...` to operate on that
//     host.  The "default" context is the local daemon.
//
// Architecture
// ────────────
//   Provider  → talks to N hosts via docker contexts
//   Node      → identifies one container by name + the host it lives on
//
// Cross-host networking comes from a single overlay network ("kind") that
// every node container attaches to.  Containers find each other through
// Swarm's embedded DNS, exactly like single-host bridge mode.
package swarm

import (
	"fmt"
	"os/exec"
	"strings"
)

// JoinSpec is everything a worker host needs to join the swarm.
type JoinSpec struct {
	ManagerAddr string // "172.16.193.6:2377"
	Token       string // SWMTKN-1-...
	Network     string // overlay network name, default "kind"
}

// Host identifies one machine in the swarm: a docker context name to
// operate on its daemon, plus an externally-reachable address to write
// into kubeconfig.
type Host struct {
	Context string // docker context name (use "default" for local)
	Addr    string // hostname or IP reachable from outside the swarm
}

// dockerArgs prefixes a docker command with --context=<name>.  Using it
// consistently means the same code path works for the manager (default
// context = local daemon) and for workers (SSH/TCP-based contexts).
func dockerArgs(ctxName string, args ...string) []string {
	return append([]string{"--context", ctxName}, args...)
}

// EnsureSwarm initialises a Swarm on the manager host (idempotent) and
// returns the JoinSpec that workers need.  If the host is already part
// of a swarm, the existing token is returned.
func EnsureSwarm(manager Host, network string) (JoinSpec, error) {
	if network == "" {
		network = "kind"
	}

	// 1. Check current state.
	out, err := exec.Command("docker",
		dockerArgs(manager.Context, "info", "--format", "{{.Swarm.LocalNodeState}}")...,
	).Output()
	if err != nil {
		return JoinSpec{}, fmt.Errorf("docker info on %s: %w", manager.Context, err)
	}
	state := strings.TrimSpace(string(out))

	// 2. If not active, init.
	if state != "active" {
		fmt.Printf("[swarm] docker swarm init on %s (advertise=%s)\n", manager.Context, manager.Addr)
		out, err := exec.Command("docker",
			dockerArgs(manager.Context, "swarm", "init", "--advertise-addr", manager.Addr)...,
		).CombinedOutput()
		if err != nil {
			return JoinSpec{}, fmt.Errorf("swarm init: %v\n%s", err, out)
		}
	}

	// 3. Pull the worker join token.
	tokenOut, err := exec.Command("docker",
		dockerArgs(manager.Context, "swarm", "join-token", "worker", "-q")...,
	).Output()
	if err != nil {
		return JoinSpec{}, fmt.Errorf("swarm join-token: %w", err)
	}
	token := strings.TrimSpace(string(tokenOut))

	return JoinSpec{
		ManagerAddr: fmt.Sprintf("%s:2377", manager.Addr),
		Token:       token,
		Network:     network,
	}, nil
}

// BuildJoinInstruction returns the shell command another host should run
// to join the swarm as a worker.  Useful for displaying or scripting.
func BuildJoinInstruction(spec JoinSpec) string {
	return fmt.Sprintf("docker swarm join --token %s %s", spec.Token, spec.ManagerAddr)
}

// RunJoinOnWorker executes the join command on a remote worker host.
// Idempotent: if the host is already part of the swarm, returns nil.
func RunJoinOnWorker(worker Host, spec JoinSpec) error {
	out, err := exec.Command("docker",
		dockerArgs(worker.Context,
			"swarm", "join", "--token", spec.Token, spec.ManagerAddr)...,
	).CombinedOutput()
	if err == nil {
		fmt.Printf("[swarm] %s joined the swarm\n", worker.Context)
		return nil
	}
	if strings.Contains(string(out), "already part of a swarm") {
		fmt.Printf("[swarm] %s already in swarm, ok\n", worker.Context)
		return nil
	}
	return fmt.Errorf("join %s: %v\n%s", worker.Context, err, out)
}

// EnsureOverlay creates the cross-host overlay network that node
// containers attach to.  Idempotent.
func EnsureOverlay(manager Host, name string) error {
	if name == "" {
		name = "kind"
	}
	if exec.Command("docker",
		dockerArgs(manager.Context, "network", "inspect", name)...,
	).Run() == nil {
		return nil
	}
	out, err := exec.Command("docker",
		dockerArgs(manager.Context,
			"network", "create", "-d", "overlay", "--attachable", name)...,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create overlay %s: %v\n%s", name, err, out)
	}
	fmt.Printf("[swarm] overlay network %q created on %s\n", name, manager.Context)
	return nil
}

// Bootstrap is a convenience wrapper that does EnsureSwarm +
// RunJoinOnWorker(for each worker) + EnsureOverlay.  Most callers will
// invoke this once at the start of a multi-host session.
func Bootstrap(manager Host, workers []Host, network string) (JoinSpec, error) {
	spec, err := EnsureSwarm(manager, network)
	if err != nil {
		return spec, err
	}
	for _, w := range workers {
		if err := RunJoinOnWorker(w, spec); err != nil {
			return spec, err
		}
	}
	if err := EnsureOverlay(manager, spec.Network); err != nil {
		return spec, err
	}
	return spec, nil
}
