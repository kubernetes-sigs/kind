/*
Copyright 2018 The Kubernetes Authors.

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

package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// kubeadmJoinAction implements action for joining nodes
// to a Kubernetes cluster.
type kubeadmJoinAction struct{}

func init() {
	registerAction("join", newKubeadmJoinAction)
}

// newKubeadmJoinAction returns a new KubeadmJoinAction
func newKubeadmJoinAction() Action {
	return &kubeadmJoinAction{}
}

// Tasks returns the list of action tasks
func (b *kubeadmJoinAction) Tasks() []Task {
	return []Task{
		{
			// Run kubeadm join on the secondary control plane Nodes
			Description: "Joining control-plane node to Kubernetes ☸",
			TargetNodes: selectSecondaryControlPlaneNodes,
			Run:         runKubeadmJoinControlPlane,
		},
		{
			// Run kubeadm join on the Worker Nodes
			Description: "Joining worker node to Kubernetes ☸",
			TargetNodes: selectWorkerNodes,
			Run:         runKubeadmJoin,
		},
	}
}

// runKubeadmJoinControlPlane executes kubadm join --control-plane command
func runKubeadmJoinControlPlane(ec *execContext, configNode *NodeReplica) error {

	// get the join address
	joinAddress, err := getJoinAddress(ec)
	if err != nil {
		// TODO(bentheelder): logging here
		return err
	}

	// get the target node for this task (the joining node)
	node, ok := ec.NodeFor(configNode)
	if !ok {
		return errors.Errorf("unable to get the handle for operating on node: %s", configNode.Name)
	}

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
	}
	if ec.ExternalEtcd() == nil {
		fileNames = append(fileNames, "etcd/ca.crt", "etcd/ca.key")
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
	controlPlaneHandle, ok := ec.NodeFor(ec.BootStrapControlPlane())
	if !ok {
		return errors.Errorf("unable to get the handle for operating on node: %s", ec.BootStrapControlPlane().Name)
	}

	// copies certificates from the bootstrap control plane node to the joining node
	for _, fileName := range fileNames {
		// sets the path of the certificate into a node
		containerPath := filepath.Join("/etc/kubernetes/pki", fileName)
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

	// run kubeadm join --control-plane
	cmd := node.Command(
		"kubeadm", "join",
		// the join command uses the docker ip and a well know port that
		// are accessible only inside the docker network
		joinAddress,
		// set the node to join as control-plane
		"--experimental-control-plane",
		// uses a well known token and skips ca certification for automating TLS bootstrap process
		"--token", kubeadm.Token,
		"--discovery-token-unsafe-skip-ca-verification",
		// preflight errors are expected, in particular for swap being enabled
		// TODO(bentheelder): limit the set of acceptable errors
		"--ignore-preflight-errors=all",
		kubeadmVerbosityFlag,
	)
	lines, err := exec.CombinedOutputLines(cmd)
	log.Debug(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to join a control plane node with kubeadm")
	}

	return nil
}

// runKubeadmJoin executes kubadm join command
func runKubeadmJoin(ec *execContext, configNode *NodeReplica) error {
	// get the join address
	joinAddress, err := getJoinAddress(ec)
	if err != nil {
		// TODO(bentheelder): logging here
		return err
	}

	// get the target node for this task (the joining node)
	node, ok := ec.NodeFor(configNode)
	if !ok {
		return errors.Errorf("unable to get the handle for operating on node: %s", configNode.Name)
	}

	// run kubeadm join
	cmd := node.Command(
		"kubeadm", "join",
		// the join command uses the docker ip and a well know port that
		// are accessible only inside the docker network
		joinAddress,
		// uses a well known token and skipping ca certification for automating TLS bootstrap process
		"--token", kubeadm.Token,
		"--discovery-token-unsafe-skip-ca-verification",
		// preflight errors are expected, in particular for swap being enabled
		// TODO(bentheelder): limit the set of acceptable errors
		"--ignore-preflight-errors=all",
		kubeadmVerbosityFlag,
	)
	lines, err := exec.CombinedOutputLines(cmd)
	log.Debug(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to join node with kubeadm")
	}

	return nil
}

// getJoinAddress return the join address thas is the control plane endpoint in case the cluster has
// an external load balancer in front of the control-plane nodes, otherwise the address of the
// boostrap control plane node.
func getJoinAddress(ec *execContext) (string, error) {
	// get the control plane endpoint, in case the cluster has an external load balancer in
	// front of the control-plane nodes
	controlPlaneEndpoint, err := getControlPlaneEndpoint(ec)
	if err != nil {
		// TODO(bentheelder): logging here
		return "", err
	}

	// if the control plane endpoint is defined we are using it as a join address
	if controlPlaneEndpoint != "" {
		return controlPlaneEndpoint, nil
	}

	// otherwise, gets the BootStrapControlPlane node
	controlPlaneHandle, ok := ec.NodeFor(ec.BootStrapControlPlane())
	if !ok {
		return "", errors.Errorf("unable to get the handle for operating on node: %s", ec.BootStrapControlPlane().Name)
	}

	// gets the IP of the bootstrap control plane node
	controlPlaneIP, err := controlPlaneHandle.IP()
	if err != nil {
		return "", errors.Wrap(err, "failed to get IP for node")
	}

	return fmt.Sprintf("%s:%d", controlPlaneIP, kubeadm.APIServerPort), nil
}
