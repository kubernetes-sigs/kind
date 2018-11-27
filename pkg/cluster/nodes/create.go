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
	"sigs.k8s.io/kind/pkg/docker"
)

// FromID creates a node handle from the node (container's) ID
func FromID(id string) *Node {
	return &Node{
		nameOrID: id,
	}
}

// helper used to get a free TCP port for the API server
func getPort() (int, error) {
	dummyListener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer dummyListener.Close()
	port := dummyListener.Addr().(*net.TCPAddr).Port
	return port, nil
}

// CreateControlPlaneNode `docker run`s the node image, note that due to
// images/node/entrypoint being the entrypoint, this container will
// effectively be paused until we call actuallyStartNode(...)
func CreateControlPlaneNode(name, image, clusterLabel string) (handle *Node, port int, err error) {
	port, err = getPort()
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get port for API server")
	}
	runArgs := []string{
		"-d", // run the container detached
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
		// publish selected port for the API server
		"--expose", fmt.Sprintf("%d", port),
		"-p", fmt.Sprintf("%d:%d", port, port),
		// explicitly set the entrypoint
		"--entrypoint=/usr/local/bin/entrypoint",
	}

	if docker.UsernsRemap() {
		// We need this argument in order to make this command work
		// in systems that have userns-remap enabled on the docker daemon
		runArgs = append(runArgs, "--userns=host")
	}

	id, err := docker.Run(
		image,
		runArgs,
		[]string{
			// explicitly pass the entrypoint argument
			"/sbin/init",
		},
	)
	// if there is a returned ID then we did create a container
	// we should return a handle so the caller can clean it up
	// we'll return a handle with the nice name though
	if id != "" {
		handle = &Node{
			nameOrID: name,
		}
	}
	if err != nil {
		return handle, 0, errors.Wrap(err, "docker run error")
	}
	return handle, port, nil
}
