package podman

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/kind/pkg/exec"
)

// newPodmanCmd returns a new exec.Cmd for podman.
func newPodmanCmd(args ...string) exec.Cmd {
	args = appendPodmanRuntimeArg(args...)
	return exec.Command("podman", args...)
}

// newPodmanCmdWithContext returns a new exec.Cmd for podman with the given context.
func newPodmanCmdWithContext(ctx context.Context, args ...string) exec.Cmd {
	args = appendPodmanRuntimeArg(args...)
	return exec.CommandContext(ctx, "podman", args...)
}

// appendPodmanRuntimeArg use the KIND_PODMAN_RUNTIME environment variable if set.
func appendPodmanRuntimeArg(args ...string) []string {
	runtime := os.Getenv("KIND_PODMAN_RUNTIME")
	if runtime != "" {
		args = append(args, fmt.Sprintf("--runtime=%s", runtime))
	}
	return args
}
