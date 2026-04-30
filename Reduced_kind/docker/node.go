package docker

import (
	"os"
	"os/exec"
	"strings"
)

// Node is the single-host implementation of cluster.Node.
//
// All methods are stateless `docker <verb>` calls keyed by name; nothing is
// cached.  This is identical to kind's design (see pkg/cluster/internal/
// providers/docker/node.go).
type Node struct {
	name string
}

// Name returns the container name.
func (n *Node) Name() string { return n.name }

// Role reads the io.x-k8s.kind.role label from the container.
func (n *Node) Role() string {
	out, _ := exec.Command(
		"docker", "inspect",
		"--format", `{{ index .Config.Labels "io.x-k8s.kind.role" }}`,
		n.name,
	).Output()
	return strings.TrimSpace(string(out))
}

// IP returns the container's IP on the kind network.
func (n *Node) IP() (string, error) {
	out, err := exec.Command(
		"docker", "inspect",
		"--format", `{{ .NetworkSettings.Networks.kind.IPAddress }}`,
		n.name,
	).Output()
	return strings.TrimSpace(string(out)), err
}

// Exec runs a command inside the container.  The privileged flag is
// required because kind runs systemd inside the container and needs to
// remount things during kubeadm.
func (n *Node) Exec(cmd string, args ...string) ([]byte, error) {
	full := append([]string{"exec", "--privileged", n.name, cmd}, args...)
	return exec.Command("docker", full...).CombinedOutput()
}

// ExecStream runs a command inside the container with stdout/stderr
// piped to the caller's terminal in real time.  Use this for long-running
// operations like `kubeadm init` so the user sees progress.
func (n *Node) ExecStream(cmd string, args ...string) error {
	full := append([]string{"exec", "--privileged", n.name, cmd}, args...)
	c := exec.Command("docker", full...)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	return c.Run()
}

// WriteFile pipes content into `cp /dev/stdin <path>` inside the container.
// Using cp instead of `tee` preserves binary content untouched.
func (n *Node) WriteFile(path, content string) error {
	cmd := exec.Command("docker", "exec", "-i", n.name, "cp", "/dev/stdin", path)
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}
