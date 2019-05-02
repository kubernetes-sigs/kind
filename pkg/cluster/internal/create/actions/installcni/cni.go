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

// Package installcni implements the install CNI action
package installcni

import (
	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

type action struct{}

// NewAction returns a new action for installing default CNI
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Installing CNI 🔌")
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

	// install the CNI network plugin
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

	// mark success
	ctx.Status.End(true)
	return nil
}
