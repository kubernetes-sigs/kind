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

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/kubeadm"
	"sigs.k8s.io/kind/pkg/kustomize"
)

// KubeadmConfigAction implements action for creating the kubadm config
// and deployng it on the bootrap control-plane node.
type KubeadmConfigAction struct{}

func init() {
	RegisterAction("config", NewKubeadmConfigAction)
}

// NewKubeadmConfigAction returns a new KubeadmConfigAction
func NewKubeadmConfigAction() Action {
	return &KubeadmConfigAction{}
}

// Tasks returns the list of action tasks
func (b *KubeadmConfigAction) Tasks() []Task {
	return []Task{
		{
			// Creates the kubeadm config file on the BootstrapControlPlaneNode
			Description: "Creating the kubeadm config file ⛵",
			TargetNodes: SelectBootstrapControlPlaneNode,
			Run:         runKubeadmConfig,
		},
	}
}

// runKubeadmConfig creates a kubeadm config file locally and then
// copies it to the node
func runKubeadmConfig(ec *execContext, configNode *config.Node) error {
	// get the target node for this task
	node, ok := ec.NodeFor(configNode)
	if !ok {
		return fmt.Errorf("unable to get the handle for operating on node: %s", configNode.Name)
	}

	// get installed kubernetes version from the node image
	kubeVersion, err := node.KubeVersion()
	if err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to get kubernetes version from node: %v")
	}

	// create kubeadm config file writing a local temp file
	kubeadmConfig, err := createKubeadmConfig(
		ec.config,
		kubeadm.ConfigData{
			ClusterName:       ec.name,
			KubernetesVersion: kubeVersion,
			APIBindPort:       kubeadm.APIServerPort,
			Token:             kubeadm.Token,
			// TODO(fabriziopandini): when external load-balancer will be
			//		implemented also controlPlaneAddress should be added
		},
	)
	if err != nil {
		// TODO(bentheelder): logging here
		return fmt.Errorf("failed to create kubeadm config: %v", err)
	}

	// defer deletion of the local temp file
	defer os.Remove(kubeadmConfig)

	// copy the config to the node
	if err := node.CopyTo(kubeadmConfig, "/kind/kubeadm.conf"); err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to copy kubeadm config to node")
	}

	return nil
}

// createKubeadmConfig creates the kubeadm config file for the cluster
// by running data through the template and writing it to a temp file
// the config file path is returned, this file should be removed later
func createKubeadmConfig(cfg *config.Config, data kubeadm.ConfigData) (path string, err error) {
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
		cfg.BootStrapControlPlane().KubeadmConfigPatches,
		cfg.BootStrapControlPlane().KubeadmConfigPatchesJSON6902,
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
