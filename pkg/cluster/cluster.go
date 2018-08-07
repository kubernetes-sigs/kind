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

// Package cluster implements kind local cluster management
package cluster

import (
	"fmt"

	"k8s.io/test-infra/kind/pkg/exec"
)

// ClusterLabelKey is applied to each "node" docker container for identification
const ClusterLabelKey = "io.k8s.kind-cluster"

// Config contains cluster options
type Config struct {
	clusterID string
	// TODO(bentheelder): fill this in
}

// NewConfig returns a new Config with default settings,
// if clusterID is "" the cluster ID will be "1"
func NewConfig(clusterID string) Config {
	// TODO(bentheelder): validation
	if clusterID == "" {
		clusterID = "1"
	}
	return Config{
		clusterID: clusterID,
	}
}

// Context contains Config and is used to create / manipulate
// kubernetes-in-docker clusters
type Context struct {
	Config
	// TODO(bentheelder): fill this in
}

// NewContext returns a new cluster management context with Config config
func NewContext(config Config) *Context {
	return &Context{
		Config: config,
	}
}

// Create provisions and starts a kubernetes-in-docker cluster
func (c *Context) Create() error {
	// TODO(bentheelder): more advanced provisioning
	// TODO(bentheelder): multiple nodes
	return c.provisionNode()
}

// Delete tears down a kubernetes-in-docker cluster
func (c *Context) Delete() error {
	// TODO(bentheelder): find and delete nodes
	nodes, err := c.ListNodes(true)
	if err != nil {
		return fmt.Errorf("error listing nodes: %v", err)
	}
	return c.deleteNodes(nodes...)
}

func (c *Context) provisionNode() error {
	// TODO(bentheelder): multiple nodes...
	nodeName := "kind-" + c.Config.clusterID + "-master"
	// create the "node" container (docker run, but it is paused, see createNode)
	if err := c.createNode(nodeName); err != nil {
		return err
	}

	// systemd-in-a-container should have read only /sys
	// https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
	// however, we need other things from `docker run --privileged` ...
	// and this flag also happens to make /sys rw, amongst other things
	if err := c.runOnNode(nodeName, []string{
		"mount", "-o", "remount,ro", "/sys",
	}); err != nil {
		// TODO(bentheelder): logging here
		c.deleteNodes(nodeName)
		return err
	}
	// TODO(bentheelder): insert other provisioning here
	// (eg enabling / disabling units, installing kube...)

	// signal the node to boot into systemd
	if err := c.actuallyStartNode(nodeName); err != nil {
		// TODO(bentheelder): logging here
		c.deleteNodes(nodeName)
		return err
	}

	return nil
}

func (c *Config) clusterLabel() string {
	return fmt.Sprintf("%s=%s", ClusterLabelKey, c.clusterID)
}

// createNode `docker run`s the node image, note that due to
// images/node/entrypoint being the entrypoint, this container will
// effectively be paused until we call actuallyStartNode(...)
func (c *Context) createNode(name string) error {
	// TODO(bentheelder): use config
	// TODO(bentheelder): logging
	// TODO(bentheelder): many of these flags should be derived from the config
	cmd := exec.Command("docker", "run")
	cmd.Args = append(cmd.Args,
		"-d", // run the container detached
		"-t", // we need a pseudo-tty for systemd logs
		// running containers in a container requires privileged
		// NOTE: we could try to replicate this with --cap-add, and use less
		// privileges, but this flag also changes some mounts that are necessary
		// including some ones docker would otherwise do by default.
		// for now this is what we want. in the future we may revisit this.
		"--privileged",
		"--security-opt", "seccomp=unconfined", // also ignore seccomp
		"--tmpfs", "/tmp", // various things depend on working /tmp
		"--tmpfs", "/run", // systemd wants a writable /run
		// docker in docker needs this, so as not to stack overlays
		"--tmpfs", "/var/lib/docker:exec",
		//"-v", "/sys/fs/cgroup:/sys/fs/cgroup:ro",
		// some k8s things want /lib/modules
		"-v", "/lib/modules:/lib/modules:ro",
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the cluster ID
		"--label", c.Config.clusterLabel(),
		"kind-node", // use our image, TODO: make this configurable
	)
	// TODO(bentheelder): collect output instead of connecting these
	cmd.InheritOutput = true
	return cmd.Run()
}

func (c *Context) deleteNodes(names ...string) error {
	cmd := exec.Command("docker", "rm")
	cmd.Args = append(cmd.Args,
		"-f", // force the container to be delete now
	)
	cmd.Args = append(cmd.Args, names...)
	return cmd.Run()
}

// runOnNode execs command on the named node
func (c *Context) runOnNode(nameOrID string, command []string) error {
	// TODO(bentheelder): use config
	// TODO(bentheelder): logging
	cmd := exec.Command("docker", "exec")
	cmd.Args = append(cmd.Args,
		"-t",           // use a tty so we can get output
		"--privileged", // run with priliges so we can remount etc..
		nameOrID,       // ... against the "node" container
	)
	cmd.Args = append(cmd.Args,
		command..., // finally, run the command supplied by the user
	)
	// TODO(bentheelder): collect output instead of connecting these
	cmd.InheritOutput = true
	return cmd.Run()
}

// signal our entrypoint (images/node/entrypoint) to boot
func (c *Context) actuallyStartNode(name string) error {
	// TODO(bentheelder): use config
	// TODO(bentheelder): logging
	cmd := exec.Command("docker", "kill")
	cmd.Args = append(cmd.Args,
		"-s", "SIGUSR1",
		name,
	)
	// TODO(bentheelder): collect output instead of connecting these
	cmd.InheritOutput = true
	return cmd.Run()
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
func (c *Context) ListNodes(alsoStopped bool) (containerIDs []string, err error) {
	cmd := exec.Command("docker", "ps")
	cmd.Args = append(cmd.Args,
		// quiet output for parsing
		"-q",
		// filter for nodes with the cluster label
		"--filter", "label="+c.Config.clusterLabel(),
	)
	// optionally show nodes that are stopped
	if alsoStopped {
		cmd.Args = append(cmd.Args, "-a")
	}
	return cmd.CombinedOutputLines()
}
