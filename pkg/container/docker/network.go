package docker

import (
	"strings"

	"sigs.k8s.io/kind/pkg/exec"
)

// NetworkInspect displays detailed information on one or more networks
func NetworkInspect(networkNames []string, format string) ([]string, error) {
	cmd := exec.Command("docker", "network", "inspect",
		"-f", format,
		strings.Join(networkNames, " "),
	)
	return exec.CombinedOutputLines(cmd)
}
