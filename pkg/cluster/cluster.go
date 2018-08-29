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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"k8s.io/test-infra/kind/pkg/cluster/kubeadm"
	"k8s.io/test-infra/kind/pkg/exec"
)

// ClusterLabelKey is applied to each "node" docker container for identification
const ClusterLabelKey = "io.k8s.test-infra.kind-cluster"

// Context is used to create / manipulate kubernetes-in-docker clusters
type Context struct {
	Name string
}

// similar to valid docker container names, but since we will prefix
// and suffix this name, we can relax it a little
// see NewContext() for usage
// https://godoc.org/github.com/docker/docker/daemon/names#pkg-constants
var validNameRE = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// NewContext returns a new cluster management context
// if name is "" the default ("1") will be used
func NewContext(name string) (ctx *Context, err error) {
	if name == "" {
		name = "1"
	}
	// validate the name
	if !validNameRE.MatchString(name) {
		return nil, fmt.Errorf(
			"'%s' is not a valid cluster name, cluster names must match `%s`",
			name, validNameRE.String(),
		)
	}
	return &Context{
		Name: name,
	}, nil
}

// ClusterLabel returns the docker object label that will be applied
// to cluster "node" containers
func (c *Context) ClusterLabel() string {
	return fmt.Sprintf("%s=%s", ClusterLabelKey, c.Name)
}

// ClusterName returns the Kubernetes cluster name based on the context name
// currently this is .Name prefixed with "kind-"
func (c *Context) ClusterName() string {
	return fmt.Sprintf("kind-%s", c.Name)
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Context) KubeConfigPath() string {
	// TODO(bentheelder): Windows?
	// configDir matches the standard directory expected by kubectl etc
	configDir := filepath.Join(os.Getenv("HOME"), ".kube")
	// note that the file name however does not, we do not want to overwite
	// the standard config, though in the future we may (?) merge them
	fileName := fmt.Sprintf("kind-config-%s", c.Name)
	return filepath.Join(configDir, fileName)
}

// Create provisions and starts a kubernetes-in-docker cluster
func (c *Context) Create(config *CreateConfig) error {
	// validate config first
	if err := config.Validate(); err != nil {
		return err
	}

	// TODO(bentheelder): multiple nodes ...
	kubeadmConfig, err := c.provisionControlPlane(
		fmt.Sprintf("kind-%s-control-plane", c.Name),
		config,
	)

	// clean up the kubeadm config file
	// NOTE: in the future we will use this for other nodes first
	if kubeadmConfig != "" {
		defer os.Remove(kubeadmConfig)
	}

	if err != nil {
		return err
	}

	log.Infof(
		"You can now use the cluster with:\n\nexport KUBECONFIG=\"%s\"\nkubectl cluster-info",
		c.KubeConfigPath(),
	)

	return nil
}

// Delete tears down a kubernetes-in-docker cluster
func (c *Context) Delete() error {
	nodes, err := c.ListNodes(true)
	if err != nil {
		return fmt.Errorf("error listing nodes: %v", err)
	}
	return c.deleteNodes(nodes...)
}

// provisionControlPlane provisions the control plane node
// and the cluster kubeadm config
func (c *Context) provisionControlPlane(nodeName string, config *CreateConfig) (kubeadmConfigPath string, err error) {
	// create the "node" container (docker run, but it is paused, see createNode)
	node, err := createNode(nodeName, c.ClusterLabel())
	if err != nil {
		return "", err
	}

	// systemd-in-a-container should have read only /sys
	// https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
	// however, we need other things from `docker run --privileged` ...
	// and this flag also happens to make /sys rw, amongst other things
	if err := node.Run("mount", "-o", "remount,ro", "/sys"); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return "", err
	}

	// signal the node entrypoint to continue booting into systemd
	if err := node.SignalStart(); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return "", err
	}

	// wait for docker to be ready
	if !node.WaitForDocker(time.Now().Add(time.Second * 30)) {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return "", fmt.Errorf("timed out waiting for docker to be ready on node")
	}

	// load the docker image artifacts into the docker daemon
	node.LoadImages()

	// get installed kubernetes version from the node image
	kubeVersion, err := node.KubeVersion()
	if err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return "", fmt.Errorf("failed to get kubernetes version from node: %v", err)
	}

	// create kubeadm config file
	kubeadmConfig, err := c.createKubeadmConfig(
		config.KubeadmConfigTemplate,
		kubeadm.ConfigData{
			ClusterName:       c.ClusterName(),
			KubernetesVersion: kubeVersion,
		},
	)

	// copy the config to the node
	if err := node.CopyTo(kubeadmConfig, "/kind/kubeadm.conf"); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return kubeadmConfig, errors.Wrap(err, "failed to copy kubeadm config to node")
	}

	// run kubeadm
	if err := node.Run(
		// init because this is the control plane node
		"kubeadm", "init",
		// preflight errors are expected, in particular for swap being enabled
		// TODO(bentheelder): limit the set of acceptable errors
		"--ignore-preflight-errors=all",
		// specify our generated config file
		"--config=/kind/kubeadm.conf",
	); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return kubeadmConfig, errors.Wrap(err, "failed to init node with kubeadm")
	}

	// set up the $KUBECONFIG
	kubeConfigPath := c.KubeConfigPath()
	if err = node.WriteKubeConfig(kubeConfigPath); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return kubeadmConfig, errors.Wrap(err, "failed to get kubeconfig from node")
	}

	// TODO(bentheelder): support other overlay networks
	if err = node.Run(
		"/bin/sh", "-c",
		`kubectl apply --kubeconfig=/etc/kubernetes/admin.conf -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"`,
	); err != nil {
		return kubeadmConfig, errors.Wrap(err, "failed to apply overlay network")
	}

	// if we are only provisioning one node, remove the master taint
	// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#master-isolation
	if config.NumNodes == 1 {
		if err = node.Run(
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
			"taint", "nodes", "--all", "node-role.kubernetes.io/master-",
		); err != nil {
			return kubeadmConfig, errors.Wrap(err, "failed to remove master taint")
		}
	}

	return kubeadmConfig, nil
}

// createKubeadmConfig creates the kubeadm config file for the cluster
// by running data through the template and writing it to a temp file
// the config file path is returned, this file should be removed later
func (c *Context) createKubeadmConfig(template string, data kubeadm.ConfigData) (path string, err error) {
	// create kubeadm config file
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeadm config")
	}
	path = f.Name()
	// generate the config contents
	config, err := kubeadm.Config(template, data)
	if err != nil {
		os.Remove(path)
		return "", err
	}
	log.Infof("Using KubeadmConfig:\n\n%s\n", config)
	_, err = f.WriteString(config)
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func (c *Context) deleteNodes(names ...string) error {
	cmd := exec.Command("docker", "rm")
	cmd.Args = append(cmd.Args,
		"-f", // force the container to be delete now
	)
	cmd.Args = append(cmd.Args, names...)
	return cmd.Run()
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
func (c *Context) ListNodes(alsoStopped bool) (containerIDs []string, err error) {
	cmd := exec.Command("docker", "ps")
	cmd.Args = append(cmd.Args,
		// quiet output for parsing
		"-q",
		// filter for nodes with the cluster label
		"--filter", "label="+c.ClusterLabel(),
	)
	// optionally list nodes that are stopped
	if alsoStopped {
		cmd.Args = append(cmd.Args, "-a")
	}
	return cmd.CombinedOutputLines()
}
