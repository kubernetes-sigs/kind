package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	kindexec "sigs.k8s.io/kind/pkg/exec"
)

// KIND CONSTANTS
const fixedNetworkName = "kind"
const clusterLabelKey = "io.x-k8s.kind.cluster"
const nodeRoleLabelKey = "io.x-k8s.kind.role"

func createContainer(name string, args []string) error {
	if err := exec.Command("docker", append([]string{"run", "--name", name}, args...)...).Run(); err != nil {
		return err
	}
	return nil
}

func deleteContainer(name string) error {
	if err := exec.Command("docker", []string{"rm", "-f", name}...).Run(); err != nil {
		return err
	}
	return nil
}

func containerIsRunning(name string) bool {
	cmd := exec.Command("docker", []string{"ps", "-q", "-f", "name=" + name}...)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return false
	}
	return true
}

func containerExist(name string) bool {
	err := exec.Command("docker", []string{"inspect", name}...).Run()
	return err == nil
}

func containerSignal(name string, signal string) error {
	err := exec.Command("docker", []string{"kill", "-s", signal, name}...).Run()
	return err
}

func execContainer(name string, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	args := []string{"exec", "--privileged"}
	if stdin != nil {
		args = append(args, "-i")
	}
	args = append(args, name)
	args = append(args, command...)
	cmd := exec.Command("docker", args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	return cmd.Run()
}

func containerIPs(name string) (ipv4 string, ipv6 string, err error) {
	// retrieve the IP address of the node using docker inspect
	cmd := kindexec.Command("docker", "inspect",
		"-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}},{{.GlobalIPv6Address}}{{end}}",
		name, // ... against the "node" container
	)
	lines, err := kindexec.OutputLines(cmd)
	if err != nil {
		return "", "", fmt.Errorf("failed to get container details: %w", err)
	}
	if len(lines) != 1 {
		return "", "", fmt.Errorf("file should only be one line, got %d lines: %w", len(lines), err)
	}
	ips := strings.Split(lines[0], ",")
	if len(ips) != 2 {
		return "", "", fmt.Errorf("container addresses should have 2 values, got %d values", len(ips))
	}
	return ips[0], ips[1], nil
}
