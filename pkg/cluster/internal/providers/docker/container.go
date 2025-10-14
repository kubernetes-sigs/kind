/*
Copyright 2025 The Kubernetes Authors.

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

package docker

import (
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/fs"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// ContainerCreationOptions holds options for creating containers
type ContainerCreationOptions struct {
	ClusterName      string
	NetworkName      string
	AllNodeNames     []string
	APIServerPort    int32
	APIServerAddress string
	IPFamily         config.ClusterIPFamily
	ClusterConfig    *config.Cluster // Full cluster config for commonArgs
}

// getNetworkName returns the docker network name to use, handling the experimental override
func getNetworkName() string {
	networkName := fixedNetworkName
	if n := os.Getenv("KIND_EXPERIMENTAL_DOCKER_NETWORK"); n != "" {
		networkName = n
	}
	return networkName
}

// prepareNodeConfig prepares a node configuration for container creation
func prepareNodeConfig(nodeConfig *config.Node) (*config.Node, error) {
	// Create a copy to avoid modifying the original
	node := nodeConfig.DeepCopy()

	// Apply defaults to the node config (including default image if not specified)
	config.SetDefaultsNode(node)

	// Handle relative paths for mounts
	for m := range node.ExtraMounts {
		hostPath := node.ExtraMounts[m].HostPath
		if !fs.IsAbs(hostPath) {
			absHostPath, err := filepath.Abs(hostPath)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to resolve absolute path for hostPath: %q", hostPath)
			}
			node.ExtraMounts[m].HostPath = absHostPath
		}
	}

	return node, nil
}

// generateRunArgsForNodeCreation generates container run arguments for a node
func generateRunArgsForNodeCreation(opts ContainerCreationOptions, node *config.Node, nodeName string) ([]string, error) {
	// Use the full cluster config if available, otherwise create minimal one
	clusterConfig := opts.ClusterConfig
	if clusterConfig == nil {
		// Create minimal cluster config with networking settings
		clusterConfig = &config.Cluster{
			Name:  opts.ClusterName,
			Nodes: []config.Node{*node},
			Networking: config.Networking{
				IPFamily: opts.IPFamily,
			},
		}
	}

	// Get common args
	genericArgs, err := commonArgs(opts.ClusterName, clusterConfig, opts.NetworkName, opts.AllNodeNames)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate common container args")
	}

	// Handle role-specific configuration
	switch node.Role {
	case config.ControlPlaneRole:
		// Add API server port mapping for control plane nodes
		node.ExtraPortMappings = append(node.ExtraPortMappings,
			config.PortMapping{
				ListenAddress: opts.APIServerAddress,
				HostPort:      opts.APIServerPort,
				ContainerPort: common.APIServerInternalPort,
			},
		)
	case config.WorkerRole:
		// Worker nodes don't need special port mappings
	default:
		return nil, errors.Errorf("unsupported node role: %q", node.Role)
	}

	// Generate run arguments
	return runArgsForNode(node, opts.IPFamily, nodeName, genericArgs)
}
