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

// Package kubeadminit implements the kubeadm init action
package kubeadminit

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"
)

// kubeadmInitAction implements action for executing the kubadm init
// and a set of default post init operations like e.g. install the
// CNI network plugin.
type action struct{}

// NewAction returns a new action for kubeadm init
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Starting control-plane 🕹️")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// get the target node for this task
	node, err := nodes.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return err
	}

	// run kubeadm
	cmd := node.Command(
		// init because this is the control plane node
		"kubeadm", "init",
		// preflight errors are expected, in particular for swap being enabled
		// TODO(bentheelder): limit the set of acceptable errors
		"--ignore-preflight-errors=all",
		// specify our generated config file
		"--config=/kind/kubeadm.conf",
		"--skip-token-print",
		// increase verbosity for debugging
		"--v=6",
	)
	lines, err := exec.CombinedOutputLines(cmd)
	log.Debug(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to init node with kubeadm")
	}

	// copies the kubeconfig files locally in order to make the cluster
	// usable with kubectl.
	// the kubeconfig file created by kubeadm internally to the node
	// must be modified in order to use the random host port reserved
	// for the API server and exposed by the node

	// retrives the random host where the API server is exposed
	// TODO(fabrizio pandini): when external load-balancer will be
	//      implemented this should be modified accordingly
	hostPort, err := node.Ports(kubeadm.APIServerPort)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	kubeConfigPath := ctx.ClusterContext.KubeConfigPath()
	if err := node.WriteKubeConfig(kubeConfigPath, hostPort); err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	// install the CNI network plugin
	// TODO(bentheelder): this should possibly be a different action?
	// TODO(bentheelder): support other overlay networks
	// first probe for a pre-installed manifest
	haveDefaultCNIManifest := true
	if err := node.Command("test", "-f", "/kind/manifests/default-cni.yaml").Run(); err != nil {
		haveDefaultCNIManifest = false
	}
	if haveDefaultCNIManifest {
		// we found the default manifest, install that
		// the images should already be loaded along with kubernetes
		if err := node.Command(
			"kubectl", "create", "--kubeconfig=/etc/kubernetes/admin.conf",
			"-f", "/kind/manifests/default-cni.yaml",
		).Run(); err != nil {
			return errors.Wrap(err, "failed to apply overlay network")
		}
	} else {
		// fallback to our old pattern of installing weave using their recommended method
		if err := node.Command(
			"/bin/sh", "-c",
			`kubectl apply --kubeconfig=/etc/kubernetes/admin.conf -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version --kubeconfig=/etc/kubernetes/admin.conf | base64 | tr -d '\n')"`,
		).Run(); err != nil {
			return errors.Wrap(err, "failed to apply overlay network")
		}
	}

	// if we are only provisioning one node, remove the master taint
	// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#master-isolation
	if len(allNodes) == 1 {
		if err := node.Command(
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
			"taint", "nodes", "--all", "node-role.kubernetes.io/master-",
		).Run(); err != nil {
			return errors.Wrap(err, "failed to remove master taint")
		}
	}

	// add the default storage class
	// TODO(bentheelder): this should possibly be a different action?
	if err := addDefaultStorageClass(node); err != nil {
		return errors.Wrap(err, "failed to add default storage class")
	}

	// mark success
	ctx.Status.End(true)
	return nil
}

// a default storage class
// we need this for e2es (StatefulSet)
const defaultStorageClassManifest = `# host-path based default storage class
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  namespace: kube-system
  name: standard
  annotations:
    storageclass.beta.kubernetes.io/is-default-class: "true"
  labels:
    addonmanager.kubernetes.io/mode: EnsureExists
provisioner: kubernetes.io/host-path`

func addDefaultStorageClass(controlPlane *nodes.Node) error {
	in := strings.NewReader(defaultStorageClassManifest)
	cmd := controlPlane.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf", "apply", "-f", "-",
	)
	cmd.SetStdin(in)
	return cmd.Run()
}
