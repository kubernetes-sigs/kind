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
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/internal/haproxy"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/kustomize"
)

// kubeadmConfigAction implements action for creating the kubadm config
// and deployng it on the bootrap control-plane node.
type kubeadmConfigAction struct{}

func init() {
	registerAction("config", newKubeadmConfigAction)
}

// NewKubeadmConfigAction returns a new KubeadmConfigAction
func newKubeadmConfigAction() Action {
	return &kubeadmConfigAction{}
}

// Tasks returns the list of action tasks
func (b *kubeadmConfigAction) Tasks() []Task {
	return []Task{
		{
			// Creates the kubeadm config file on the BootstrapControlPlaneNode
			Description: "Creating the kubeadm config file ⛵",
			TargetNodes: selectBootstrapControlPlaneNode,
			Run:         runKubeadmConfig,
		},
	}
}

// runKubeadmConfig creates a kubeadm config file locally and then
// copies it to the node
func runKubeadmConfig(ec *execContext, configNode *NodeReplica) error {
	// get the target node for this task
	node, ok := ec.NodeFor(configNode)
	if !ok {
		return errors.Errorf("unable to get the handle for operating on node: %s", configNode.Name)
	}

	// get installed kubernetes version from the node image
	kubeVersion, err := node.KubeVersion()
	if err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to get kubernetes version from node: %v")
	}

	// get the control plane endpoint, in case the cluster has an external load balancer in
	// front of the control-plane nodes
	controlPlaneEndpoint, err := getControlPlaneEndpoint(ec)
	if err != nil {
		// TODO(bentheelder): logging here
		return err
	}

	// create kubeadm config file writing a local temp file
	kubeadmConfig, err := createKubeadmConfig(
		ec.Config,
		ec.DerivedConfig,
		kubeadm.ConfigData{
			ClusterName:          ec.Name(),
			KubernetesVersion:    kubeVersion,
			ControlPlaneEndpoint: controlPlaneEndpoint,
			APIBindPort:          kubeadm.APIServerPort,
			Token:                kubeadm.Token,
		},
	)
	defer os.Remove(kubeadmConfig)
	if err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to create kubeadm config")
	}

	// copy the config to the node
	if err := node.CopyTo(kubeadmConfig, "/kind/kubeadm.conf"); err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to copy kubeadm config to node")
	}

	return nil
}

// getControlPlaneEndpoint return the control plane endpoint in case the cluster has an external load balancer in
// front of the control-plane nodes, otherwise return an empty string.
func getControlPlaneEndpoint(ec *execContext) (string, error) {
	if ec.ExternalLoadBalancer() == nil {
		return "", nil
	}

	// gets the handle for the load balancer node
	loadBalancerHandle, ok := ec.NodeFor(ec.ExternalLoadBalancer())
	if !ok {
		return "", errors.Errorf("unable to get the handle for operating on node: %s", ec.ExternalLoadBalancer().Name)
	}

	// gets the IP of the load balancer
	loadBalancerIP, err := loadBalancerHandle.IP()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get IP for node: %s", ec.ExternalLoadBalancer().Name)
	}

	return fmt.Sprintf("%s:%d", loadBalancerIP, haproxy.ControlPlanePort), nil
}

// createKubeadmConfig creates the kubeadm config file for the cluster
// by running data through the template and writing it to a temp file
// the config file path is returned, this file should be removed later
func createKubeadmConfig(cfg *config.Config, derived *DerivedConfig, data kubeadm.ConfigData) (path string, err error) {
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
	// fix all the patches to have name metadata matching the generated config
	patches, jsonPatches := setPatchNames(
		derived.BootStrapControlPlane().KubeadmConfigPatches,
		derived.BootStrapControlPlane().KubeadmConfigPatchesJSON6902,
	)
	// apply patches
	// TODO(bentheelder): this does not respect per node patches at all
	// either make patches cluster wide, or change this
	patchedConfig, err := kustomize.Build([]string{config}, patches, jsonPatches)
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

// setPatchNames sets the targeted object name on every patch to be the fixed
// name we use when generating config objects (we have one of each type, all of
// which have the same fixed name)
func setPatchNames(patches []string, jsonPatches []kustomize.PatchJSON6902) ([]string, []kustomize.PatchJSON6902) {
	fixedPatches := make([]string, len(patches))
	fixedJSONPatches := make([]kustomize.PatchJSON6902, len(jsonPatches))
	for i, patch := range patches {
		// insert the generated name metadata
		fixedPatches[i] = fmt.Sprintf("metadata:\nname: %s\n%s", kubeadm.ObjectName, patch)
	}
	for i, patch := range jsonPatches {
		// insert the generated name metadata
		patch.Name = kubeadm.ObjectName
		fixedJSONPatches[i] = patch
	}
	return fixedPatches, fixedJSONPatches
}
