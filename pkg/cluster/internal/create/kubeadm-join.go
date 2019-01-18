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
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kind/pkg/exec"
	"strings"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
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
func runKubeadmJoin(ec *execContext, configNode *NodeReplica) error {
	// before running join, it should be retrived

	// gets the node where
	// TODO(fabrizio pandini): when external load-balancer will be
	//      implemented this should be modified accordingly
	controlPlaneHandle, ok := ec.NodeFor(ec.DerivedConfig.BootStrapControlPlane())
	if !ok {
		return fmt.Errorf("unable to get the handle for operating on node: %s", ec.DerivedConfig.BootStrapControlPlane().Name)
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

	line, err := kubeadm.ParseTemplate(strings.TrimSpace(ec.Context.Config.Kubeadm.JoinCommand),
		struct {
			Token string
			IP    string
			Port  int
		}{
			kubeadm.Token,
			controlPlaneIP,
			kubeadm.APIServerPort,
		})
	if err != nil {
		return fmt.Errorf("error when parsing kubeadm join template: %v", err)
	}

	// run kubeadm
	cmd := string(line)
	args := strings.Split(cmd, " ")
	log.Debugf("Kubeadm init command: %s", cmd)
	if err := exec.RunLoggingOutputOnFail(node.Command(args[0], args[1:]...)); err != nil {
		return errors.Wrapf(err, "failed to join node with '%s'", cmd)
	}

	// TODO(fabrizio pandini): might be we want to run post-kubeadm hooks on workers too

	// TODO(fabrizio pandini): might be we want to run post-setup hooks on workers too

	return nil
}
