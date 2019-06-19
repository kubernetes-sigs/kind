/*
Copyright 2019 The Kubernetes Authors.

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

// Package kubeadmjoin implements the kubeadm join action
package kubeadmjoin

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/concurrent"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// Action implements action for creating the kubeadm join
// and deployng it on the bootrap control-plane node.
type Action struct{}

// NewAction returns a new action for creating the kubeadm jion
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// join secondary control plane nodes if any
	secondaryControlPlanes, err := nodes.SecondaryControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	if len(secondaryControlPlanes) > 0 {
		if err := joinSecondaryControlPlanes(
			ctx, allNodes, secondaryControlPlanes,
		); err != nil {
			return err
		}
	}

	// then join worker nodes if any
	workers, err := nodes.SelectNodesByRole(allNodes, constants.WorkerNodeRoleValue)
	if err != nil {
		return err
	}
	if len(workers) > 0 {
		if err := joinWorkers(ctx, allNodes, workers); err != nil {
			return err
		}
	}

	return nil
}

func joinSecondaryControlPlanes(
	ctx *actions.ActionContext,
	allNodes []nodes.Node,
	secondaryControlPlanes []nodes.Node,
) error {
	ctx.Status.Start("Joining more control-plane nodes 🎮")
	defer ctx.Status.End(false)

	// TODO(bentheelder): it's too bad we can't do this concurrently
	// (this is not safe currently)
	for _, node := range secondaryControlPlanes {
		if err := runKubeadmJoinControlPlane(ctx, allNodes, &node); err != nil {
			return err
		}
	}

	ctx.Status.End(true)
	return nil
}

func joinWorkers(
	ctx *actions.ActionContext,
	allNodes []nodes.Node,
	workers []nodes.Node,
) error {
	ctx.Status.Start("Joining worker nodes 🚜")
	defer ctx.Status.End(false)

	// create the workers concurrently
	fns := []func() error{}
	for _, node := range workers {
		node := node // capture loop variable
		fns = append(fns, func() error {
			return runKubeadmJoin(ctx, allNodes, &node)
		})
	}
	if err := concurrent.UntilError(fns); err != nil {
		return err
	}

	ctx.Status.End(true)
	return nil
}

// runKubeadmJoinControlPlane executes kubadm join --control-plane command
func runKubeadmJoinControlPlane(
	ctx *actions.ActionContext,
	allNodes []nodes.Node,
	node *nodes.Node,
) error {
	// creates the folder tree for pre-loading necessary cluster certificates
	// on the joining node
	if err := node.Command("mkdir", "-p", "/etc/kubernetes/pki/etcd").Run(); err != nil {
		return errors.Wrap(err, "failed to join node with kubeadm")
	}

	// define the list of necessary cluster certificates
	fileNames := []string{
		"ca.crt", "ca.key",
		"front-proxy-ca.crt", "front-proxy-ca.key",
		"sa.pub", "sa.key",
		// TODO(someone): if we gain external etcd support these will be
		// handled differently
		"etcd/ca.crt", "etcd/ca.key",
	}

	// creates a temporary folder on the host that should acts as a transit area
	// for moving necessary cluster certificates
	tmpDir, err := fs.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	err = os.MkdirAll(filepath.Join(tmpDir, "/etcd"), os.ModePerm)
	if err != nil {
		return err
	}

	// get the handle for the bootstrap control plane node (the source for necessary cluster certificates)
	controlPlaneHandle, err := nodes.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return err
	}

	// copies certificates from the bootstrap control plane node to the joining node
	for _, fileName := range fileNames {
		// sets the path of the certificate into a node
		containerPath := path.Join("/etc/kubernetes/pki", fileName)
		// set the path of the certificate into the tmp area on the host
		tmpPath := filepath.Join(tmpDir, fileName)
		// copies from bootstrap control plane node to tmp area
		if err := controlPlaneHandle.CopyFrom(containerPath, tmpPath); err != nil {
			return errors.Wrapf(err, "failed to copy certificate %s", fileName)
		}
		// copies from tmp area to joining node
		if err := node.CopyTo(tmpPath, containerPath); err != nil {
			return errors.Wrapf(err, "failed to copy certificate %s", fileName)
		}
	}

	return runKubeadmJoin(ctx, allNodes, node)
}

// runKubeadmJoin executes kubadm join command
func runKubeadmJoin(
	ctx *actions.ActionContext,
	allNodes []nodes.Node,
	node *nodes.Node,
) error {
	// run kubeadm join
	// TODO(bentheelder): this should be using the config file
	cmd := node.Command(
		"kubeadm", "join",
		// the join command uses the config file generated in a well known location
		"--config", "/kind/kubeadm.conf",
		// preflight errors are expected, in particular for swap being enabled
		// TODO(bentheelder): limit the set of acceptable errors
		"--ignore-preflight-errors=all",
		// increase verbosity for debugging
		"--v=6",
	)
	lines, err := exec.CombinedOutputLines(cmd)
	log.Debug(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to join node with kubeadm")
	}

	return nil
}
