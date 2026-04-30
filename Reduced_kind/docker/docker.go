// Package docker is the single-host Reduced_kind provider.
//
// It implements cluster.Provider and cluster.Node by shelling out to the
// `docker` CLI on the local daemon.  The two label keys below are how kind
// stores cluster membership inside Docker (no external state).
//
// Multi-host extension story
// ──────────────────────────
// Every command issued here is `docker <args>`.  To make this multi-host you
// inject a `--context=<host>` flag in front of every invocation:
//
//	docker --context=worker-1 run ...
//	docker --context=worker-1 exec ...
//
// The `Provider` would carry a list of hosts, look up `Node.Host` from the
// config, and route each command accordingly.  See DESIGN.md for details.
package docker

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"reducedkind/cluster"
	"reducedkind/config"
)

const (
	networkName  = "kind"
	clusterLabel = "io.x-k8s.kind.cluster"
	roleLabel    = "io.x-k8s.kind.role"
)

// Provider is the docker-CLI-on-local-daemon implementation.
type Provider struct{}

// New returns a Provider that talks to the local docker daemon.
func New() cluster.Provider { return &Provider{} }

// ─── Provider interface ────────────────────────────────────────────────

func (p *Provider) Provision(cfg *config.Cluster) error {
	if err := ensureNetwork(); err != nil {
		return err
	}
	for i, n := range cfg.Nodes {
		name := nodeName(cfg.Name, n.Role, i, cfg.Nodes)
		if err := runContainer(name, cfg.Name, n); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) ListNodes(clusterName string) ([]cluster.Node, error) {
	out, err := exec.Command(
		"docker", "ps", "-a",
		"--filter", fmt.Sprintf("label=%s=%s", clusterLabel, clusterName),
		"--format", "{{.Names}}",
	).Output()
	if err != nil {
		return nil, err
	}
	var nodes []cluster.Node
	for _, name := range strings.Fields(string(out)) {
		nodes = append(nodes, &Node{name: name})
	}
	return nodes, nil
}

func (p *Provider) DeleteNodes(nodes []cluster.Node) error {
	if len(nodes) == 0 {
		return nil
	}
	args := []string{"rm", "-f", "-v"}
	for _, n := range nodes {
		args = append(args, n.Name())
	}
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm: %w\n%s", err, out)
	}
	return nil
}

func (p *Provider) GetAPIServerEndpoint(clusterName string) (string, error) {
	nodes, err := p.ListNodes(clusterName)
	if err != nil {
		return "", err
	}
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return "", fmt.Errorf("no control-plane node in cluster %q", clusterName)
	}
	out, err := exec.Command("docker", "port", cp.Name(), "6443").Output()
	if err != nil {
		return "", err
	}
	// `docker port` returns lines like "0.0.0.0:38291" — take the first.
	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return line, nil
}

// ─── helpers ──────────────────────────────────────────────────────────

func ensureNetwork() error {
	if exec.Command("docker", "network", "inspect", networkName).Run() == nil {
		return nil
	}
	out, err := exec.Command("docker", "network", "create", networkName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker network create %s: %w\n%s", networkName, err, out)
	}
	return nil
}

func runContainer(name, clusterName string, n config.Node) error {
	image := n.Image
	if image == "" {
		image = config.DefaultNodeImage
	}
	args := []string{
		"run", "-d",
		"--name", name,
		"--hostname", name,
		"--privileged",
		"--restart=on-failure:1",
		"--network", networkName,
		"--label", fmt.Sprintf("%s=%s", clusterLabel, clusterName),
		"--label", fmt.Sprintf("%s=%s", roleLabel, string(n.Role)),
		"--tmpfs", "/tmp",
		"--tmpfs", "/run",
		"--volume", "/var",
		"--volume", "/lib/modules:/lib/modules:ro",
		"--cgroupns=private",
	}
	// Only the control-plane container needs to expose 6443 on the host.
	if n.Role == config.ControlPlaneRole {
		args = append(args, "-p", "127.0.0.1:0:6443")
	}
	args = append(args, image)
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run %s (image=%s): %w\n%s", name, image, err, out)
	}
	// kindest/node enables systemd + containerd at boot, but they take a few
	// seconds to come up.  kubeadm/kubectl will fail noisily if we proceed
	// before /run/containerd/containerd.sock exists.  Wait for it.
	return waitForContainerd(name, 60*time.Second)
}

// waitForContainerd polls until /run/containerd/containerd.sock exists inside
// the named container, or timeout elapses.  Mirrors kind's
// WaitUntilLogRegexpMatches but uses a direct socket check, which is simpler
// and matches what kubeadm actually needs.
func waitForContainerd(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if exec.Command("docker", "exec", name,
			"test", "-S", "/run/containerd/containerd.sock").Run() == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("containerd socket did not appear in %s within %s", name, timeout)
}

// nodeName mirrors common.MakeNodeNamer in kind:  the first node of a role
// has no numeric suffix; subsequent ones get 2, 3, ...
func nodeName(clusterName string, role config.NodeRole, idx int, all []config.Node) string {
	count := 0
	for i := 0; i <= idx; i++ {
		if all[i].Role == role {
			count++
		}
	}
	if count == 1 {
		return fmt.Sprintf("%s-%s", clusterName, role)
	}
	return fmt.Sprintf("%s-%s%d", clusterName, role, count)
}
