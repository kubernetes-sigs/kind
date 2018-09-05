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
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/test-infra/kind/pkg/cluster/kubeadm"
	"k8s.io/test-infra/kind/pkg/exec"
)

type nodeHandle struct {
	// must be one of docker container ID or name
	nameOrID string
}

// createNode `docker run`s the node image, note that due to
// images/node/entrypoint being the entrypoint, this container will
// effectively be paused until we call actuallyStartNode(...)
func createNode(name, clusterLabel string) (handle *nodeHandle, err error) {
	cmd := exec.Command("docker", "run")
	cmd.Args = append(cmd.Args,
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
		// docker in docker needs this, so as not to stack overlays
		"--tmpfs", "/var/lib/docker:exec",
		// some k8s things want /lib/modules
		"-v", "/lib/modules:/lib/modules:ro",
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the cluster ID
		"--label", clusterLabel,
		"--expose", "6443", // expose API server port
		// pick a random ephemeral port to forward to the API server
		"--publish-all",
		"kind-node", // use our image, TODO: make this configurable
	)
	cmd.Debug = true
	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	return &nodeHandle{name}, nil
}

// SignalStart sends SIGUSR1 to the node, which signals our entrypoint to boot
// see images/node/entrypoint
func (nh *nodeHandle) SignalStart() error {
	cmd := exec.Command("docker", "kill")
	cmd.Args = append(cmd.Args,
		"-s", "SIGUSR1",
		nh.nameOrID,
	)
	// TODO(bentheelder): collect output instead of connecting these
	cmd.InheritOutput = true
	return cmd.Run()
}

// Run execs command, args... on the node
func (nh *nodeHandle) Run(command string, args ...string) error {
	cmd := exec.Command("docker", "exec")
	cmd.Args = append(cmd.Args,
		"-t",           // use a tty so we can get output
		"--privileged", // run with priliges so we can remount etc..
		nh.nameOrID,    // ... against the "node" container
		command,        // with the command specified
	)
	cmd.Args = append(cmd.Args,
		args..., // finally, with the args specified
	)
	cmd.InheritOutput = true
	return cmd.Run()
}

// CombinedOutputLines execs command, args... on the node, returning the output lines
func (nh *nodeHandle) CombinedOutputLines(command string, args ...string) ([]string, error) {
	cmd := exec.Command("docker", "exec")
	cmd.Args = append(cmd.Args,
		"-t",           // use a tty so we can get output
		"--privileged", // run with priliges so we can remount etc..
		nh.nameOrID,    // ... against the "node" container
		command,        // with the command specified
	)
	cmd.Args = append(cmd.Args,
		args..., // finally, with the args specified
	)
	return cmd.CombinedOutputLines()
}

// helper to copy source file to dest on the node
func (nh *nodeHandle) CopyTo(source, dest string) error {
	cmd := exec.Command("docker", "cp")
	cmd.Args = append(cmd.Args,
		source, // from the source file
		nh.nameOrID+":"+dest, // to the node, at dest
	)
	cmd.InheritOutput = true
	return cmd.Run()
}

// WaitForDocker waits for Docker to be ready on the node
// it returns true on success, and false on a timeout
func (nh *nodeHandle) WaitForDocker(until time.Time) bool {
	return tryUntil(until, func() bool {
		out, err := nh.CombinedOutputLines("systemctl", "is-active", "docker")
		if err != nil {
			return false
		}
		return len(out) == 1 && out[0] == "active"
	})
}

// helper that calls `try()`` in a loop until the deadline `until`
// has passed or `try()`returns true, returns wether try ever returned true
func tryUntil(until time.Time, try func() bool) bool {
	now := time.Now()
	for until.After(now) {
		if try() {
			return true
		}
	}
	return false
}

// LoadImages loads image tarballs stored on the node into docker on the node
func (nh *nodeHandle) LoadImages() {
	// load images cached on the node into docker
	if err := nh.Run(
		"find",
		"/kind/images",
		"-name", "*.tar",
		"-exec", "docker", "load", "-i", "{}", ";",
	); err != nil {
		log.Warningf("Failed to preload docker images: %v", err)
		return
	}
	// retag images that are missing -amd64 as image:tag -> image-amd64:tag
	// bazel built images are currently missing these
	// TODO(bentheelder): this is a bit gross, move this logic out of bash
	if err := nh.Run(
		"/bin/bash", "-c",
		`docker images --format='{{.Repository}}:{{.Tag}}' | grep -v amd64 | xargs -L 1 -I '{}' /bin/bash -c 'docker tag "{}" "$(echo "{}" | sed s/:/-amd64:/)"'`,
	); err != nil {
		log.Warningf("Failed to re-tag docker images: %v", err)
	}

	nh.Run("docker", "images")
}

// KubeVersion returns the Kubernetes version installed on the node
func (nh *nodeHandle) KubeVersion() (version string, err error) {
	// grab kubernetes version from the node image
	lines, err := nh.CombinedOutputLines("cat", "/kind/version")
	if err != nil {
		return "", errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return "", fmt.Errorf("file should only be one line, got %d lines", len(lines))
	}
	return lines[0], nil
}

// matches kubeconfig server entry like:
//    server: https://172.17.0.2:6443
// which we rewrite to:
//    server: https://localhost:$PORT
var serverAddressRE = regexp.MustCompile(`^(\s+server:) https://.*:\d+$`)

// WriteKubeConfig writes a fixed KUBECONFIG to dest
// this should only be called on a control plane node
func (nh *nodeHandle) WriteKubeConfig(dest string) error {
	// get the forwarded api server port
	port, err := nh.GetForwardedPort(kubeadm.APIServerPort)
	if err != nil {
		return err
	}

	lines, err := nh.CombinedOutputLines("cat", "/etc/kubernetes/admin.conf")
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	// fix the config file, swapping out the server for the forwarded localhost:port
	var buff bytes.Buffer
	for _, line := range lines {
		match := serverAddressRE.FindStringSubmatch(line)
		if len(match) > 1 {
			line = fmt.Sprintf("%s https://localhost:%d", match[1], port)
		}
		buff.WriteString(line)
		buff.WriteString("\n")
	}

	return ioutil.WriteFile(dest, buff.Bytes(), 0600)
}

// GetForwardedPort takes the port number within the "node" container
// and returns the port it was forwarded to ouside the container
func (nh *nodeHandle) GetForwardedPort(port uint16) (uint16, error) {
	cmd := exec.Command("docker", "port")
	cmd.Args = append(cmd.Args,
		nh.nameOrID,             // ports are looked up by container
		fmt.Sprintf("%d", port), // limit to the port we are looking up
	)
	lines, err := cmd.CombinedOutputLines()
	if err != nil {
		return 0, err
	}
	if len(lines) != 1 {
		return 0, fmt.Errorf("invalid output: %v", lines)
	}
	parts := strings.Split(lines[0], ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid output: %v", lines)
	}
	v, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}
