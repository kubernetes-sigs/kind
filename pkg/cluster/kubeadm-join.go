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

package cluster

import (
	"fmt"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/kubeadm"
)

// kubeadmJoinAction implements action for joining nodes
// to a Kubernetes cluster.
type kubeadmJoinAction struct{}

func init() {
	registerAction("join", newKubeadmJoinAction)
}

// newKubeadmJoinAction returns a new KubeadmJoinAction
func newKubeadmJoinAction() action {
	return &kubeadmJoinAction{}
}

// Tasks returns the list of action tasks
func (b *kubeadmJoinAction) Tasks() []task {
	return []task{
		// TODO(fabrizio pandini): add Run kubeadm join --experimental-master
		//      on SecondaryControlPlaneNodes
		{
			// Run kubeadm join on the WorkeNodes
			Description: "Joining worker node to Kubernetes ☸",
			TargetNodes: selectWorkerNodes,
			Run:         runKubeadmJoin,
		},
	}
}

// runKubeadmJoin executes kubadm join
func runKubeadmJoin(ec *execContext, configNode *nodeReplica) error {
	// before running join, it should be retrived

	// gets the node where
	// TODO(fabrizio pandini): when external load-balancer will be
	//      implemented this should be modified accordingly
	controlPlaneHandle, ok := ec.NodeFor(ec.derived.BootStrapControlPlane())
	if !ok {
		return fmt.Errorf("unable to get the handle for operating on node: %s", ec.derived.BootStrapControlPlane().Name)
	}

	// gets the IP of the bootstrap master node
	controlPlaneIP, err := controlPlaneHandle.IP()
	if err != nil {
		return errors.Wrap(err, "failed to get IP for node")
	}

	// get the target node for this task
	node, ok := ec.NodeFor(configNode)
	if !ok {
		return fmt.Errorf("unable to get the handle for operating on node: %s", configNode.Name)
	}

	// TODO(fabrizio pandini): might be we want to run pre-kubeadm hooks on workers too

	// run kubeadm
	if err := node.Command(
		"kubeadm", "join",
		// the control plane address uses the docker ip and a well know APIServerPort that
		// are accessible only inside the docker network
		fmt.Sprintf("%s:%d", controlPlaneIP, kubeadm.APIServerPort),
		// uses a well known token and skipping ca certification for automating TLS bootstrap process
		"--token", kubeadm.Token,
		"--discovery-token-unsafe-skip-ca-verification",
		// preflight errors are expected, in particular for swap being enabled
		// TODO(bentheelder): limit the set of acceptable errors
		"--ignore-preflight-errors=all",
	).Run(); err != nil {
		return errors.Wrap(err, "failed to join node with kubeadm")
	}

	// TODO(fabrizio pandini): might be we want to run post-kubeadm hooks on workers too

	// TODO(fabrizio pandini): might be we want to run post-setup hooks on workers too

	return nil
}
