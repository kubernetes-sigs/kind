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

// Package config implements the kubeadm config action
package config

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/concurrent"
	"sigs.k8s.io/kind/pkg/kustomize"
)

// Action implements action for creating the kubeadm config
// and deployng it on the bootrap control-plane node.
type Action struct{}

// NewAction returns a new action for creating the kubadm config
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Creating kubeadm config 📜")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// create the kubeadm init configuration
	node, err := nodes.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return err
	}

	// get installed kubernetes version from the node image
	kubeVersion, err := node.KubeVersion()
	if err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to get kubernetes version from node")
	}

	// get the control plane endpoint, in case the cluster has an external load balancer in
	// front of the control-plane nodes
	controlPlaneEndpoint, controlPlaneEndpointIPv6, err := nodes.GetControlPlaneEndpoint(allNodes)
	if err != nil {
		// TODO(bentheelder): logging here
		return err
	}

	// configure the right protocol addresses
	if ctx.Config.Networking.IPFamily == "ipv6" {
		controlPlaneEndpoint = controlPlaneEndpointIPv6
	}

	// create kubeadm init config
	fns := []func() error{}

	configData := kubeadm.ConfigData{
		ClusterName:          ctx.ClusterContext.Name(),
		KubernetesVersion:    kubeVersion,
		ControlPlaneEndpoint: controlPlaneEndpoint,
		APIBindPort:          kubeadm.APIServerPort,
		APIServerAddress:     ctx.Config.Networking.APIServerAddress,
		Token:                kubeadm.Token,
		PodSubnet:            ctx.Config.Networking.PodSubnet,
		ServiceSubnet:        ctx.Config.Networking.ServiceSubnet,
		ControlPlane:         true,
		IPv6:                 ctx.Config.Networking.IPFamily == "ipv6",
	}

	fns = append(fns, func() error {
		return writeKubeadmConfig(ctx.Config, configData, node)
	})

	// create the kubeadm join configuration for secondary control plane nodes if any
	secondaryControlPlanes, err := nodes.SecondaryControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	if len(secondaryControlPlanes) > 0 {
		// create the workers concurrently
		for _, node := range secondaryControlPlanes {
			node := node             // capture loop variable
			configData := configData // copy config data
			fns = append(fns, func() error {
				return writeKubeadmConfig(ctx.Config, configData, &node)
			})
		}
	}

	// then create the kubeadm join config for the worker nodes if any
	workers, err := nodes.SelectNodesByRole(allNodes, constants.WorkerNodeRoleValue)
	if err != nil {
		return err
	}
	if len(workers) > 0 {
		// create the workers concurrently
		for _, node := range workers {
			node := node             // capture loop variable
			configData := configData // copy config data
			configData.ControlPlane = false
			fns = append(fns, func() error {
				return writeKubeadmConfig(ctx.Config, configData, &node)
			})
		}
	}

	// Create the config in all nodes concurrently
	if err := concurrent.UntilError(fns); err != nil {
		return err
	}

	// mark success
	ctx.Status.End(true)
	return nil
}

// getKubeadmConfig generates the kubeadm config contents for the cluster
// by running data through the template.
func getKubeadmConfig(cfg *config.Cluster, data kubeadm.ConfigData) (path string, err error) {
	// generate the config contents
	config, err := kubeadm.Config(data)
	if err != nil {
		return "", err
	}
	// fix all the patches to have name metadata matching the generated config
	patches, jsonPatches := setPatchNames(
		allPatchesFromConfig(cfg),
	)
	// apply patches
	// TODO(bentheelder): this does not respect per node patches at all
	// either make patches cluster wide, or change this
	patched, err := kustomize.Build([]string{config}, patches, jsonPatches)
	if err != nil {
		return "", err
	}
	return removeMetadata(patched), nil
}

// trims out the metadata.name we put in the config for kustomize matching,
// kubeadm will complain about this otherwise
func removeMetadata(kustomized string) string {
	return strings.Replace(
		kustomized,
		`metadata:
  name: config
`,
		"",
		-1,
	)
}

func allPatchesFromConfig(cfg *config.Cluster) (patches []string, jsonPatches []kustomize.PatchJSON6902) {
	return cfg.KubeadmConfigPatches, cfg.KubeadmConfigPatchesJSON6902
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

// writeKubeadmConfig writes the kubeadm configuration in the specified node
func writeKubeadmConfig(cfg *config.Cluster, data kubeadm.ConfigData, node *nodes.Node) error {
	// get the node ip address
	nodeAddress, nodeAddressIPv6, err := node.IP()
	if err != nil {
		return errors.Wrap(err, "failed to get IP for node")
	}

	data.NodeAddress = nodeAddress
	// configure the right protocol addresses
	if cfg.Networking.IPFamily == "ipv6" {
		data.NodeAddress = nodeAddressIPv6
	}

	kubeadmConfig, err := getKubeadmConfig(cfg, data)

	if err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to generate kubeadm config content")
	}

	log.Debug("Using kubeadm config:\n" + kubeadmConfig)

	// copy the config to the node
	if err := node.WriteFile("/kind/kubeadm.conf", kubeadmConfig); err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to copy kubeadm config to node")
	}

	return nil
}
