package swarm

import (
	"os"
	"os/exec"
	"strings"
)

// Node satisfies cluster.Node by routing every docker invocation through
// `--context=<host>`.  That single flag is the only difference from the
// single-host docker.Node implementation; everything else (the kubeadm
// commands, kubectl applies, etc.) runs identically because it goes
// through this Node.Exec/WriteFile abstraction.
type Node struct {
	name string // container name, e.g. "test-control-plane"
	host string // docker context name, e.g. "ecotype-35"
}

func (n *Node) Name() string { return n.name }

// Role reads the kind role label off the container.
func (n *Node) Role() string {
	out, _ := exec.Command("docker",
		dockerArgs(n.host, "inspect",
			"--format", `{{ index .Config.Labels "io.x-k8s.kind.role" }}`,
			n.name)...,
	).Output()
	return strings.TrimSpace(string(out))
}

// IP returns the node container's address on the kind overlay network.
func (n *Node) IP() (string, error) {
	out, err := exec.Command("docker",
		dockerArgs(n.host, "inspect",
			"--format", `{{ .NetworkSettings.Networks.kind.IPAddress }}`,
			n.name)...,
	).Output()
	return strings.TrimSpace(string(out)), err
}

// Exec runs a command inside the container on its host.
func (n *Node) Exec(cmd string, args ...string) ([]byte, error) {
	full := dockerArgs(n.host, "exec", "--privileged", n.name, cmd)
	full = append(full, args...)
	return exec.Command("docker", full...).CombinedOutput()
}

// ExecStream runs the command with stdout/stderr piped to the caller's
// terminal.  Used for long-running operations like `kubeadm init`.
func (n *Node) ExecStream(cmd string, args ...string) error {
	full := dockerArgs(n.host, "exec", "--privileged", n.name, cmd)
	full = append(full, args...)
	c := exec.Command("docker", full...)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	return c.Run()
}

// WriteFile creates a file inside the container by piping content
// through `cp /dev/stdin`.
func (n *Node) WriteFile(path, content string) error {
	cmd := exec.Command("docker",
		dockerArgs(n.host, "exec", "-i", n.name, "cp", "/dev/stdin", path)...,
	)
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}
