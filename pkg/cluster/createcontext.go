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
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/docker"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// createContext is a superset of Context implementing helpers internal to Context.Create()
type createContext struct {
	*Context
	status *logutil.Status
	config *config.Config
	*derivedConfig
	retain           bool          // if we should retain nodes after failing to create.
	waitForReady     time.Duration // Wait for the control plane node to be ready.
	ControlPlaneMeta *ControlPlaneMeta
}

// EnsureNodeImages ensures that the node images used by the create
// configuration are present
func (cc *createContext) EnsureNodeImages() {
	var images = map[string]bool{}

	// For all the nodes defined in the `kind` config
	for _, configNode := range cc.AllReplicas() {
		if _, ok := images[configNode.Image]; ok {
			continue
		}

		// prints user friendly message
		image := configNode.Image
		if strings.Contains(image, "@sha256:") {
			image = strings.Split(image, "@sha256:")[0]
		}
		cc.status.Start(fmt.Sprintf("Ensuring node image (%s) 🖼", image))

		// attempt to explicitly pull the image if it doesn't exist locally
		// we don't care if this errors, we'll still try to run which also pulls
		_, _ = docker.PullIfNotPresent(configNode.Image, 4)

		// marks the images as already pulled
		images[configNode.Image] = true
	}
}

// provisionNodes takes care of creating all the containers
// that will host `kind` nodes
func (cc *createContext) provisionNodes() (nodeList map[string]*nodes.Node, err error) {
	nodeList = map[string]*nodes.Node{}

	// For all the nodes defined in the `kind` config
	for _, configNode := range cc.AllReplicas() {

		cc.status.Start(fmt.Sprintf("[%s] Creating node container 📦", configNode.Name))
		// create the node into a container (docker run, but it is paused, see createNode)
		var name = fmt.Sprintf("kind-%s-%s", cc.name, configNode.Name)
		var node *nodes.Node

		switch configNode.Role {
		case config.ControlPlaneRole:
			node, err = nodes.CreateControlPlaneNode(name, configNode.Image, cc.ClusterLabel())
		case config.WorkerRole:
			node, err = nodes.CreateWorkerNode(name, configNode.Image, cc.ClusterLabel())
		}
		if err != nil {
			return nodeList, err
		}
		nodeList[configNode.Name] = node

		cc.status.Start(fmt.Sprintf("[%s] Fixing mounts 🗻", configNode.Name))
		// we need to change a few mounts once we have the container
		// we'd do this ahead of time if we could, but --privileged implies things
		// that don't seem to be configurable, and we need that flag
		if err := node.FixMounts(); err != nil {
			// TODO(bentheelder): logging here
			return nodeList, err
		}

		cc.status.Start(fmt.Sprintf("[%s] Starting systemd 🖥", configNode.Name))
		// signal the node container entrypoint to continue booting into systemd
		if err := node.SignalStart(); err != nil {
			// TODO(bentheelder): logging here
			return nodeList, err
		}

		cc.status.Start(fmt.Sprintf("[%s] Waiting for docker to be ready 🐋", configNode.Name))
		// wait for docker to be ready
		if !node.WaitForDocker(time.Now().Add(time.Second * 30)) {
			// TODO(bentheelder): logging here
			return nodeList, fmt.Errorf("timed out waiting for docker to be ready on node")
		}

		// load the docker image artifacts into the docker daemon
		cc.status.Start(fmt.Sprintf("[%s] Pre-loading images 🐋", configNode.Name))
		node.LoadImages()

	}

	return nodeList, nil
}
