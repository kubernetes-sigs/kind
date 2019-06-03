package runtime

import (
	"github.com/pkg/errors"
	"os/exec"
)

// Preflight runs checks to make sure that the container runtime
// is working as expected
func Preflight() error {
	checks := []func() error{
		dockerIsRunning,
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return errors.Wrap(err, "preflight check failed")
		}
	}

	return nil
}

// dockerIsRunning asserts that the docker daemon is running and is responsive
func dockerIsRunning() error {
	err := exec.Command("docker", "ps").Run()
	if err != nil {
		return errors.Wrap(err, "could not connect to a Docker daemon")
	}
	return nil
}
