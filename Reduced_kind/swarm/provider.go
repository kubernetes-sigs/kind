package swarm

import (
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"time"

	"reducedkind/cluster"
	"reducedkind/config"
)

// Provider implements cluster.Provider by sharding docker calls across
// the configured hosts.  hosts[0] is the manager (used for swarm-wide
// operations like overlay creation and as the host that runs the
// control-plane container by default).
type Provider struct {
	hosts   []Host
	network string // overlay name, default "kind"
}

// New returns a multi-host Provider.  The first host is treated as the
// manager.  Network is the overlay name (defaults to "kind" if empty).
func New(hosts []Host, network string) cluster.Provider {
	if network == "" {
		network = "kind"
	}
	return &Provider{hosts: hosts, network: network}
}

func (p *Provider) manager() Host  { return p.hosts[0] }
func (p *Provider) workers() []Host { return p.hosts[1:] }

// pickHostForNode decides which host gets the i-th node.  Strategy:
//   - if config.Node.Host names a known context, use it;
//   - otherwise round-robin across all hosts (manager first).
func (p *Provider) pickHostForNode(n config.Node, idx int) Host {
	if n.Host != "" {
		for _, h := range p.hosts {
			if h.Context == n.Host {
				return h
			}
		}
	}
	return p.hosts[idx%len(p.hosts)]
}

// hostByContext finds a Host by its docker context name.
func (p *Provider) hostByContext(ctx string) (Host, bool) {
	for _, h := range p.hosts {
		if h.Context == ctx {
			return h, true
		}
	}
	return Host{}, false
}

// ─── cluster.Provider interface ───────────────────────────────────────

// Provision creates one container per cfg.Node, distributed across the
// configured hosts.  Containers attach to the kind overlay so they can
// reach each other regardless of host.
func (p *Provider) Provision(cfg *config.Cluster) error {
	if len(p.hosts) == 0 {
		return fmt.Errorf("no hosts configured")
	}
	// Make sure the overlay exists on the manager (idempotent).
	if err := EnsureOverlay(p.manager(), p.network); err != nil {
		return err
	}

	namer := newNodeNamer(cfg.Name)
	for i, n := range cfg.Nodes {
		h := p.pickHostForNode(n, i)
		name := namer(string(n.Role))
		if err := p.runNodeContainer(h, name, cfg.Name, n); err != nil {
			return err
		}
		if err := waitForContainerd(h, name, 60*time.Second); err != nil {
			return err
		}
	}
	return nil
}

// ListNodes fans out `docker ps` to every host and aggregates the
// results.  Each returned Node remembers which host it lives on.
func (p *Provider) ListNodes(clusterName string) ([]cluster.Node, error) {
	var all []cluster.Node
	for _, h := range p.hosts {
		out, err := exec.Command("docker",
			dockerArgs(h.Context,
				"ps", "-a",
				"--filter", fmt.Sprintf("label=io.x-k8s.kind.cluster=%s", clusterName),
				"--format", "{{.Names}}")...,
		).Output()
		if err != nil {
			return nil, fmt.Errorf("ps on host %s: %w", h.Context, err)
		}
		for _, name := range strings.Fields(string(out)) {
			all = append(all, &Node{name: name, host: h.Context})
		}
	}
	// Stable order (control-plane first, then workers by name) makes
	// callers like cluster.FindByRole behave deterministically.
	sort.Slice(all, func(i, j int) bool { return all[i].Name() < all[j].Name() })
	return all, nil
}

// DeleteNodes groups nodes by host and rm -f -v on each daemon.
func (p *Provider) DeleteNodes(nodes []cluster.Node) error {
	if len(nodes) == 0 {
		return nil
	}
	byHost := map[string][]string{}
	for _, n := range nodes {
		sn, ok := n.(*Node)
		if !ok {
			return fmt.Errorf("DeleteNodes: not a swarm.Node: %T", n)
		}
		byHost[sn.host] = append(byHost[sn.host], sn.name)
	}
	for ctx, names := range byHost {
		args := dockerArgs(ctx, "rm", "-f", "-v")
		args = append(args, names...)
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("rm on host %s: %v\n%s", ctx, err, out)
		}
	}
	return nil
}

// GetAPIServerEndpoint returns "<host-addr>:<mapped-port>" for the
// control-plane container — which is what kubectl on the user's
// machine needs to reach the API server.
func (p *Provider) GetAPIServerEndpoint(clusterName string) (string, error) {
	nodes, err := p.ListNodes(clusterName)
	if err != nil {
		return "", err
	}
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return "", fmt.Errorf("no control-plane node in cluster %q", clusterName)
	}
	sn := cp.(*Node)

	host, ok := p.hostByContext(sn.host)
	if !ok {
		return "", fmt.Errorf("control-plane is on host %q which isn't in the configured host list", sn.host)
	}

	// docker port returns lines like "0.0.0.0:38291" — first line is enough.
	out, err := exec.Command("docker",
		dockerArgs(sn.host, "port", sn.name, "6443")...,
	).Output()
	if err != nil {
		return "", fmt.Errorf("docker port on %s: %w", sn.host, err)
	}
	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	_, port, err := net.SplitHostPort(line)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", line, err)
	}
	return net.JoinHostPort(host.Addr, port), nil
}

// ─── helpers (private) ────────────────────────────────────────────────

// runNodeContainer issues `docker --context=<host> run -d ...` with the
// same flag set as the single-host provider.  The crucial difference is
// `--network=<overlay>` instead of the local bridge.
func (p *Provider) runNodeContainer(h Host, name, clusterName string, n config.Node) error {
	image := n.Image
	if image == "" {
		image = config.DefaultNodeImage
	}
	args := dockerArgs(h.Context,
		"run", "-d",
		"--name", name,
		"--hostname", name,
		"--privileged",
		"--security-opt", "seccomp=unconfined",
		"--security-opt", "apparmor=unconfined",
		"--init=false",
		"--restart=on-failure:1",
		"--network", p.network,
		"--label", fmt.Sprintf("io.x-k8s.kind.cluster=%s", clusterName),
		"--label", fmt.Sprintf("io.x-k8s.kind.role=%s", string(n.Role)),
		"--tmpfs", "/tmp",
		"--tmpfs", "/run",
		"--volume", "/var",
		"--volume", "/lib/modules:/lib/modules:ro",
		"--cgroupns=private",
	)
	if n.Role == config.ControlPlaneRole {
		// Expose the API server on the host so external kubectl can reach it.
		args = append(args, "-p", "0.0.0.0:0:6443")
	}
	args = append(args, image)
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("run %s on %s: %v\n%s", name, h.Context, err, out)
	}
	fmt.Printf("[swarm] started %s on %s\n", name, h.Context)
	return nil
}

func waitForContainerd(h Host, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if exec.Command("docker",
			dockerArgs(h.Context, "exec", name, "test", "-S", "/run/containerd/containerd.sock")...,
		).Run() == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("containerd socket did not appear in %s on %s within %s",
		name, h.Context, timeout)
}

// newNodeNamer returns a function that produces deterministic node
// names: <cluster>-<role> for the first of each role, then <role>2,
// <role>3, ... for subsequent ones.  Same convention as the single-host
// provider.
func newNodeNamer(clusterName string) func(role string) string {
	counter := map[string]int{}
	return func(role string) string {
		counter[role]++
		if counter[role] == 1 {
			return fmt.Sprintf("%s-%s", clusterName, role)
		}
		return fmt.Sprintf("%s-%s%d", clusterName, role, counter[role])
	}
}
