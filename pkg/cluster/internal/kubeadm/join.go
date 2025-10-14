/*
Copyright 2025 The Kubernetes Authors.

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

package kubeadm

import (
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/internal/version"
	"sigs.k8s.io/kind/pkg/log"
)

// RunKubeadmJoin executes kubeadm join command using the config file at /kind/kubeadm.conf
func RunKubeadmJoin(logger log.Logger, node nodes.Node) error {
	kubeVersionStr, err := nodeutils.KubeVersion(node)
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version from node")
	}
	kubeVersion, err := version.ParseGeneric(kubeVersionStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse kubernetes version %q", kubeVersionStr)
	}

	args := []string{
		"join",
		// the join command uses the config file at /kind/kubeadm.conf
		"--config", "/kind/kubeadm.conf",
		// increase verbosity for debugging
		"--v=6",
	}
	// Newer versions set this in the config file.
	if kubeVersion.LessThan(version.MustParseSemantic("v1.23.0")) {
		// Skip preflight to avoid pulling images.
		// Kind pre-pulls images and preflight may conflict with that.
		args = append(args, "--skip-phases=preflight")
	}

	// run kubeadm join
	cmd := node.Command("kubeadm", args...)
	lines, err := exec.CombinedOutputLines(cmd)
	logger.V(3).Info(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to join node with kubeadm")
	}

	return nil
}
