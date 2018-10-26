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
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/kubeadm"
	"sigs.k8s.io/kind/pkg/docker"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/kustomize"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// ClusterLabelKey is applied to each "node" docker container for identification
const ClusterLabelKey = "io.k8s.sigs.kind.cluster"

// Context is used to create / manipulate kubernetes-in-docker clusters
type Context struct {
	name   string
	status *logutil.Status
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
		name: name,
	}, nil
}

// ClusterLabel returns the docker object label that will be applied
// to cluster "node" containers
func (c *Context) ClusterLabel() string {
	return fmt.Sprintf("%s=%s", ClusterLabelKey, c.name)
}

// Name returns the context's name
func (c *Context) Name() string {
	return c.name
}

// ClusterName returns the Kubernetes cluster name based on the context name
// currently this is .Name prefixed with "kind-"
func (c *Context) ClusterName() string {
	return fmt.Sprintf("kind-%s", c.name)
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Context) KubeConfigPath() string {
	// TODO(bentheelder): Windows?
	// configDir matches the standard directory expected by kubectl etc
	configDir := filepath.Join(os.Getenv("HOME"), ".kube")
	// note that the file name however does not, we do not want to overwite
	// the standard config, though in the future we may (?) merge them
	fileName := fmt.Sprintf("kind-config-%s", c.name)
	return filepath.Join(configDir, fileName)
}

// Create provisions and starts a kubernetes-in-docker cluster
func (c *Context) Create(cfg *config.Config) error {
	// validate config first
	if err := cfg.Validate(); err != nil {
		return err
	}

	fmt.Printf("Creating cluster '%s' ...\n", c.ClusterName())
	c.status = logutil.NewStatus()
	c.status.MaybeWrapLogrus(log.StandardLogger())

	defer c.status.End(false)
	c.status.Start(fmt.Sprintf("Ensuring node image (%s) 🖼", cfg.Image))

	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = docker.PullIfNotPresent(cfg.Image, 4)

	// TODO(bentheelder): multiple nodes ...
	kubeadmConfig, err := c.provisionControlPlane(
		fmt.Sprintf("kind-%s-control-plane", c.name),
		cfg,
	)

	// clean up the kubeadm config file
	// NOTE: in the future we will use this for other nodes first
	if kubeadmConfig != "" {
		defer os.Remove(kubeadmConfig)
	}
	if err != nil {
		return err
	}

	c.status.End(true)
	fmt.Printf(
		"Cluster creation complete. You can now use the cluster with:\n\nexport KUBECONFIG=\"$(kind get kubeconfig-path)\"\nkubectl cluster-info\n",
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
func (c *Context) provisionControlPlane(
	nodeName string,
	cfg *config.Config,
) (kubeadmConfigPath string, err error) {
	c.status.Start(fmt.Sprintf("[%s] Creating node container 📦", nodeName))
	// create the "node" container (docker run, but it is paused, see createNode)
	node, port, err := createControlPlaneNode(nodeName, cfg.Image, c.ClusterLabel())
	if err != nil {
		return "", err
	}

	c.status.Start(fmt.Sprintf("[%s] Fixing mounts 🗻", nodeName))
	// we need to change a few mounts once we have the container
	// we'd do this ahead of time if we could, but --privileged implies things
	// that don't seem to be configurable, and we need that flag
	if err := node.FixMounts(); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return "", err
	}

	// run any pre-boot hooks
	if cfg.NodeLifecycle != nil {
		for _, hook := range cfg.NodeLifecycle.PreBoot {
			if err := node.RunHook(&hook, "preBoot"); err != nil {
				return "", err
			}
		}
	}

	c.status.Start(fmt.Sprintf("[%s] Starting systemd 🖥", nodeName))
	// signal the node entrypoint to continue booting into systemd
	if err := node.SignalStart(); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return "", err
	}

	c.status.Start(fmt.Sprintf("[%s] Waiting for docker to be ready 🐋", nodeName))
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
		cfg,
		kubeadm.ConfigData{
			ClusterName:       c.ClusterName(),
			KubernetesVersion: kubeVersion,
			APIBindPort:       port,
		},
	)
	if err != nil {
		c.deleteNodes(node.nameOrID)
		return "", fmt.Errorf("failed to create kubeadm config: %v", err)
	}

	// copy the config to the node
	if err := node.CopyTo(kubeadmConfig, "/kind/kubeadm.conf"); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		c.deleteNodes(node.nameOrID)
		return kubeadmConfig, errors.Wrap(err, "failed to copy kubeadm config to node")
	}

	// run any pre-kubeadm hooks
	if cfg.NodeLifecycle != nil {
		for _, hook := range cfg.NodeLifecycle.PreKubeadm {
			if err := node.RunHook(&hook, "preKubeadm"); err != nil {
				return kubeadmConfig, err
			}
		}
	}

	// run kubeadm
	c.status.Start(
		fmt.Sprintf(
			"[%s] Starting Kubernetes (this may take a minute) ☸",
			nodeName,
		))
	if err := node.RunQ(
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
		return kubeadmConfig, errors.Wrap(err, "failed to init node with kubeadm")
	}

	// run any post-kubeadm hooks
	if cfg.NodeLifecycle != nil {
		for _, hook := range cfg.NodeLifecycle.PostKubeadm {
			if err := node.RunHook(&hook, "postKubeadm"); err != nil {
				return kubeadmConfig, err
			}
		}
	}

	// set up the $KUBECONFIG
	kubeConfigPath := c.KubeConfigPath()
	if err = node.WriteKubeConfig(kubeConfigPath); err != nil {
		// TODO(bentheelder): logging here
		// TODO(bentheelder): add a flag to retain the broken nodes for debugging
		return kubeadmConfig, errors.Wrap(err, "failed to get kubeconfig from node")
	}

	// TODO(bentheelder): support other overlay networks
	if err = node.RunQ(
		"/bin/sh", "-c",
		`kubectl apply --kubeconfig=/etc/kubernetes/admin.conf -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version --kubeconfig=/etc/kubernetes/admin.conf | base64 | tr -d '\n')"`,
	); err != nil {
		return kubeadmConfig, errors.Wrap(err, "failed to apply overlay network")
	}

	// if we are only provisioning one node, remove the master taint
	// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#master-isolation
	if cfg.NumNodes == 1 {
		if err = node.RunQ(
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
			"taint", "nodes", "--all", "node-role.kubernetes.io/master-",
		); err != nil {
			return kubeadmConfig, errors.Wrap(err, "failed to remove master taint")
		}
	}

	// add the default storage class
	if err := node.RunQWithInput(
		strings.NewReader(defaultStorageClassManifest),
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "apply", "-f", "-",
	); err != nil {
		return kubeadmConfig, errors.Wrap(err, "failed to add default storage class")
	}

	// run any post-overlay hooks
	if cfg.NodeLifecycle != nil {
		for _, hook := range cfg.NodeLifecycle.PostSetup {
			if err := node.RunHook(&hook, "postSetup"); err != nil {
				return kubeadmConfig, err
			}
		}
	}

	return kubeadmConfig, nil
}

// createKubeadmConfig creates the kubeadm config file for the cluster
// by running data through the template and writing it to a temp file
// the config file path is returned, this file should be removed later
func (c *Context) createKubeadmConfig(cfg *config.Config, data kubeadm.ConfigData) (path string, err error) {
	// create kubeadm config file
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubeadm config")
	}
	path = f.Name()
	// generate the config contents
	config, err := kubeadm.Config(data)
	if err != nil {
		os.Remove(path)
		return "", err
	}
	// apply patches
	patchedConfig, err := kustomize.Build(
		[]string{config},
		cfg.KubeadmConfigPatches,
		cfg.KubeadmConfigPatchesJSON6902,
	)
	if err != nil {
		os.Remove(path)
		return "", err
	}
	// write to the file
	log.Infof("Using KubeadmConfig:\n\n%s\n", patchedConfig)
	_, err = f.WriteString(patchedConfig)
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

// config has slices of string, but we want bytes for kustomize
func stringSliceToByteSliceSlice(ss []string) [][]byte {
	bss := [][]byte{}
	for _, s := range ss {
		bss = append(bss, []byte(s))
	}
	return bss
}

func (c *Context) deleteNodes(names ...string) error {
	cmd := exec.Command("docker", "rm")
	cmd.Args = append(cmd.Args,
		"-f", // force the container to be delete now
		"-v", // delete volumes
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
