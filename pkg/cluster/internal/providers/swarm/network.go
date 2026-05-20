/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package swarm

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// swarmOverlayName is the overlay network all node containers attach to.
// Mirrors docker.fixedNetworkName but lives on a swarm overlay driver
// instead of a local bridge, so VXLAN-tunnelled traffic spans hosts.
const swarmOverlayName = "kind"

// ensureSwarmOverlay creates the overlay network on the swarm manager
// if it doesn't exist.  Idempotent: a pre-existing network is fine.
func ensureSwarmOverlay(manager Host, name string) error {
	if name == "" {
		name = swarmOverlayName
	}
	if err := exec.Command("docker",
		dockerArgs(manager.Context, "network", "inspect", name)...,
	).Run(); err == nil {
		return nil
	}
	lines, err := exec.CombinedOutputLines(exec.Command("docker",
		dockerArgs(manager.Context,
			"network", "create", "-d", "overlay", "--attachable", name)...,
	))
	if err != nil {
		return errors.Wrapf(err, "create overlay %s on %s: %s",
			name, manager.Context, strings.Join(lines, "\n"))
	}
	return nil
}

// swarmActive returns true if the docker context already belongs to a
// swarm with state "active".
func swarmActive(h Host) (bool, error) {
	lines, err := exec.OutputLines(exec.Command("docker",
		dockerArgs(h.Context, "info", "--format", "{{.Swarm.LocalNodeState}}")...,
	))
	if err != nil {
		return false, errors.Wrapf(err, "docker info on %s", h.Context)
	}
	if len(lines) == 0 {
		return false, nil
	}
	return strings.TrimSpace(lines[0]) == "active", nil
}

// initSwarmIfNeeded runs `docker swarm init` on the manager when its
// swarm state is not yet active, and joins workers into the swarm.
// Skipped silently if the swarm is already up.
func initSwarmIfNeeded(manager Host, workers []Host) error {
	mgrActive, err := swarmActive(manager)
	if err != nil {
		return err
	}
	if !mgrActive {
		lines, err := exec.CombinedOutputLines(exec.Command("docker",
			dockerArgs(manager.Context, "swarm", "init",
				"--advertise-addr", manager.Addr)...,
		))
		if err != nil {
			return errors.Wrapf(err, "swarm init on %s: %s",
				manager.Context, strings.Join(lines, "\n"))
		}
	}
	tokLines, err := exec.OutputLines(exec.Command("docker",
		dockerArgs(manager.Context, "swarm", "join-token", "worker", "-q")...,
	))
	if err != nil {
		return errors.Wrap(err, "swarm join-token")
	}
	if len(tokLines) == 0 {
		return errors.New("swarm join-token returned no output")
	}
	token := strings.TrimSpace(tokLines[0])
	mgrAddr := fmt.Sprintf("%s:2377", manager.Addr)

	for _, w := range workers {
		active, err := swarmActive(w)
		if err != nil {
			return err
		}
		if active {
			continue
		}
		joinOut, err := exec.CombinedOutputLines(exec.Command("docker",
			dockerArgs(w.Context, "swarm", "join", "--token", token, mgrAddr)...,
		))
		if err != nil {
			joined := strings.Join(joinOut, "\n")
			if strings.Contains(joined, "already part of a swarm") {
				continue
			}
			return errors.Wrapf(err, "swarm join on %s: %s", w.Context, joined)
		}
	}
	return nil
}
