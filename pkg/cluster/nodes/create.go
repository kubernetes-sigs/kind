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

package nodes

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer"
	"sigs.k8s.io/kind/pkg/container/cri"
	"sigs.k8s.io/kind/pkg/container/docker"
)

// FromName creates a node handle from the node' Name
func FromName(name string) *Node {
	return &Node{
		name:  name,
		cache: &nodeCache{},
	}
}

// helper used to get a free TCP port for the API server
func getPort() (int32, error) {
	dummyListener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer dummyListener.Close()
	port := dummyListener.Addr().(*net.TCPAddr).Port
	return int32(port), nil
}

// CreateControlPlaneNode creates a contol-plane node
// and gets ready for exposing the the API server
func CreateControlPlaneNode(name, image, clusterLabel, listenAddress string, port int32, mounts []cri.Mount, portMappings []cri.PortMapping) (node *Node, err error) {
	// gets a random host port for the API server
	if port == 0 {
		p, err := getPort()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get port for API server")
		}
		port = p
	}

	// add api server port mapping
	portMappingsWithAPIServer := append(portMappings, cri.PortMapping{
		ListenAddress: listenAddress,
		HostPort:      port,
		ContainerPort: kubeadm.APIServerPort,
	})
	node, err = createNode(
		name, image, clusterLabel, constants.ControlPlaneNodeRoleValue, mounts, portMappingsWithAPIServer,
		// publish selected port for the API server
		"--expose", fmt.Sprintf("%d", port),
	)
	if err != nil {
		return node, err
	}

	// stores the port mapping into the node internal state
	node.cache.set(func(cache *nodeCache) {
		cache.ports = map[int32]int32{kubeadm.APIServerPort: port}
	})

	return node, nil
}

// CreateExternalLoadBalancerNode creates an external loab balancer node
// and gets ready for exposing the the API server and the load balancer admin console
func CreateExternalLoadBalancerNode(name, image, clusterLabel, listenAddress string, port int32) (node *Node, err error) {
	// gets a random host port for control-plane load balancer
	// gets a random host port for the API server
	if port == 0 {
		p, err := getPort()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get port for API server")
		}
		port = p
	}

	// load balancer port mapping
	portMappings := []cri.PortMapping{{
		ListenAddress: listenAddress,
		HostPort:      port,
		ContainerPort: loadbalancer.ControlPlanePort,
	}}
	node, err = createNode(name, image, clusterLabel, constants.ExternalLoadBalancerNodeRoleValue,
		nil, portMappings,
		// publish selected port for the control plane
		"--expose", fmt.Sprintf("%d", port),
	)
	if err != nil {
		return node, err
	}

	// stores the port mapping into the node internal state
	node.cache.set(func(cache *nodeCache) {
		cache.ports = map[int32]int32{loadbalancer.ControlPlanePort: port}
	})

	return node, nil
}

// CreateWorkerNode creates a worker node
func CreateWorkerNode(name, image, clusterLabel string, mounts []cri.Mount, portMappings []cri.PortMapping) (node *Node, err error) {
	node, err = createNode(name, image, clusterLabel, constants.WorkerNodeRoleValue, mounts, portMappings)
	if err != nil {
		return node, err
	}
	return node, nil
}

// TODO(bentheelder): refactor this to not have extraArgs
// createNode `docker run`s the node image, note that due to
// images/node/entrypoint being the entrypoint, this container will
// effectively be paused until we call actuallyStartNode(...)
func createNode(name, image, clusterLabel, role string, mounts []cri.Mount, portMappings []cri.PortMapping, extraArgs ...string) (handle *Node, err error) {
	runArgs := []string{
		"-d", // run the container detached
		"-t", // allocate a tty for entrypoint logs
		// running containers in a container requires privileged
		// NOTE: we could try to replicate this with --cap-add, and use less
		// privileges, but this flag also changes some mounts that are necessary
		// including some ones docker would otherwise do by default.
		// for now this is what we want. in the future we may revisit this.
		"--privileged",
		"--security-opt", "seccomp=unconfined", // also ignore seccomp
		"--tmpfs", "/tmp", // various things depend on working /tmp
		"--tmpfs", "/run", // systemd wants a writable /run
		// some k8s things want /lib/modules
		"-v", "/lib/modules:/lib/modules:ro",
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the cluster ID
		"--label", clusterLabel,
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", constants.NodeRoleKey, role),
	}

	// pass proxy environment variables to be used by node's docker deamon
	proxyDetails, err := getProxyDetails()
	if err != nil || proxyDetails == nil {
		return nil, errors.Wrap(err, "proxy setup error")
	}
	for key, val := range proxyDetails.Envs {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	// adds node specific args
	runArgs = append(runArgs, extraArgs...)

	if docker.UsernsRemap() {
		// We need this argument in order to make this command work
		// in systems that have userns-remap enabled on the docker daemon
		runArgs = append(runArgs, "--userns=host")
	}

	err = docker.Run(
		image,
		docker.WithRunArgs(runArgs...),
		docker.WithMounts(mounts),
		docker.WithPortMappings(portMappings),
	)

	// we should return a handle so the caller can clean it up
	handle = FromName(name)
	if err != nil {
		return handle, errors.Wrap(err, "docker run error")
	}

	return handle, nil
}
