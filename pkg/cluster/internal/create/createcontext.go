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
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/internal/context"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/container/docker"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Context is a superset of the main cluster Context implementing helpers internal
// to the user facing Context.Create()
// TODO(bentheelder): elminate this object
type Context struct {
	*context.Context
	// other fields
	Status *logutil.Status
	Config *config.Config
	*DerivedConfig
	Retain      bool         // if we should retain nodes after failing to create.
	ExecOptions []ExecOption // options to be forwarded to the exec command.
}

//ExecOption is an execContext configuration option supplied to Exec
type ExecOption func(*execContext)

// WaitForReady configures execContext to use interval as maximum wait time for the control plane node to be ready
func WaitForReady(interval time.Duration) ExecOption {
	return func(c *execContext) {
		c.waitForReady = interval
	}
}

// Exec actions on kubernetes-in-docker cluster
// TODO(bentheelder): refactor this further
// Actions are repetitive, high level abstractions/workflows composed
// by one or more lower level tasks, that automatically adapt to the
// current cluster topology
func (cc *Context) Exec(nodeList map[string]*nodes.Node, actions []string, options ...ExecOption) error {
	// init the exec context and logging
	ec := &execContext{
		Context: cc,
		nodes:   nodeList,
	}

	ec.status = logutil.NewStatus(os.Stdout)
	ec.status.MaybeWrapLogrus(log.StandardLogger())

	defer ec.status.End(false)

	// apply exec options
	for _, option := range options {
		option(ec)
	}

	// Create an ExecutionPlan that applies the given actions to the
	// topology defined in the config
	executionPlan, err := newExecutionPlan(ec.DerivedConfig, actions)
	if err != nil {
		return err
	}

	// Executes all the selected action
	for _, plannedTask := range executionPlan {
		ec.status.Start(fmt.Sprintf("[%s] %s", plannedTask.Node.Name, plannedTask.Task.Description))

		err := plannedTask.Task.Run(ec, plannedTask.Node)
		if err != nil {
			// in case of error, the execution plan is halted
			log.Error(err)
			return err
		}
	}
	ec.status.End(true)

	return nil
}

// EnsureNodeImages ensures that the node images used by the create
// configuration are present
func (cc *Context) EnsureNodeImages() {
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
		cc.Status.Start(fmt.Sprintf("Ensuring node image (%s) 🖼", image))

		// attempt to explicitly pull the image if it doesn't exist locally
		// we don't care if this errors, we'll still try to run which also pulls
		_, _ = docker.PullIfNotPresent(configNode.Image, 4)

		// marks the images as already pulled
		images[configNode.Image] = true
	}
}

// ProvisionNodes takes care of creating all the containers
// that will host `kind` nodes
func (cc *Context) ProvisionNodes() (nodeList map[string]*nodes.Node, err error) {
	nodeList = map[string]*nodes.Node{}

	// For all the nodes defined in the `kind` config
	for _, configNode := range cc.AllReplicas() {

		cc.Status.Start(fmt.Sprintf("[%s] Creating node container 📦", configNode.Name))
		// create the node into a container (docker run, but it is paused, see createNode)
		var name = fmt.Sprintf("%s-%s", cc.Name(), configNode.Name)
		var node *nodes.Node

		switch configNode.Role {
		case config.ExternalLoadBalancerRole:
			node, err = nodes.CreateExternalLoadBalancerNode(name, configNode.Image, cc.ClusterLabel())
		case config.ControlPlaneRole:
			node, err = nodes.CreateControlPlaneNode(name, configNode.Image, cc.ClusterLabel(), configNode.ExtraMounts)
		case config.WorkerRole:
			node, err = nodes.CreateWorkerNode(name, configNode.Image, cc.ClusterLabel(), configNode.ExtraMounts)
		}
		if err != nil {
			return nodeList, err
		}
		nodeList[configNode.Name] = node

		cc.Status.Start(fmt.Sprintf("[%s] Fixing mounts 🗻", configNode.Name))
		// we need to change a few mounts once we have the container
		// we'd do this ahead of time if we could, but --privileged implies things
		// that don't seem to be configurable, and we need that flag
		if err := node.FixMounts(); err != nil {
			// TODO(bentheelder): logging here
			return nodeList, err
		}

		cc.Status.Start(fmt.Sprintf("[%s] Configuring proxy 🐋", configNode.Name))
		if err := node.SetProxy(); err != nil {
			// TODO: logging here
			return nodeList, errors.Wrapf(err, "failed to set proxy for %s", configNode.Name)
		}

		cc.Status.Start(fmt.Sprintf("[%s] Starting systemd 🖥", configNode.Name))
		// signal the node container entrypoint to continue booting into systemd
		if err := node.SignalStart(); err != nil {
			// TODO(bentheelder): logging here
			return nodeList, err
		}

		cc.Status.Start(fmt.Sprintf("[%s] Waiting for docker to be ready 🐋", configNode.Name))
		// wait for docker to be ready
		if !node.WaitForDocker(time.Now().Add(time.Second * 30)) {
			// TODO(bentheelder): logging here
			return nodeList, errors.New("timed out waiting for docker to be ready on node")
		}

		// load the docker image artifacts into the docker daemon
		cc.Status.Start(fmt.Sprintf("[%s] Pre-loading images 🐋", configNode.Name))
		node.LoadImages()

	}

	return nodeList, nil
}
