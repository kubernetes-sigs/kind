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

// Package constants contains well known constants for kind clusters
package constants

// DefaultClusterName is the default cluster Context name
const DefaultClusterName = "kind"

// start Envoy with dynamic configuration by using files that implement the xDS protocol.
// https://www.envoyproxy.io/docs/envoy/latest/start/quick-start/configuration-dynamic-filesystem
const DynamicFilesystemConfigTemplate = `
node:
  cluster: %s
  id: %s

dynamic_resources:
  cds_config:
    resource_api_version: V3
    path_config_source:
      path: %s
  lds_config:
    resource_api_version: V3
    path_config_source:
      path: %s

admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 10000
`

// keep in sync with dynamicFilesystemConfig
const (
	ProxyConfigPathCDS = "/home/envoy/cds.yaml"
	ProxyConfigPathLDS = "/home/envoy/lds.yaml"
	ProxyConfigPath    = "/home/envoy/envoy.yaml"
	ProxyConfigDir     = "/home/envoy"
)

/* node role value constants */
const (
	// ControlPlaneNodeRoleValue identifies a node that hosts a Kubernetes
	// control-plane.
	//
	// NOTE: in single node clusters, control-plane nodes act as worker nodes
	ControlPlaneNodeRoleValue string = "control-plane"

	// WorkerNodeRoleValue identifies a node that hosts a Kubernetes worker
	WorkerNodeRoleValue string = "worker"

	// ExternalLoadBalancerNodeRoleValue identifies a node that hosts an
	// external load balancer for the API server in HA configurations.
	//
	// Please note that `kind` nodes hosting external load balancer are not
	// kubernetes nodes
	ExternalLoadBalancerNodeRoleValue string = "external-load-balancer"

	// ExternalEtcdNodeRoleValue identifies a node that hosts an external-etcd
	// instance.
	//
	// WARNING: this node type is not yet implemented!
	//
	// Please note that `kind` nodes hosting external etcd are not
	// kubernetes nodes
	ExternalEtcdNodeRoleValue string = "external-etcd"
)
