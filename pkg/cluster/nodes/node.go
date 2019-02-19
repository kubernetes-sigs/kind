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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/constants"

	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/exec"
)

// Node represents a handle to a kind node
// This struct must be created by one of: CreateControlPlane
// It should not be manually instantiated
// Node impleemnts exec.Cmder
type Node struct {
	// must be one of docker container ID or name
	nameOrID string
	// cached node info etc.
	nodeCache
}

// assert Node implements Cmder
var _ exec.Cmder = &Node{}

// Cmder returns an exec.Cmder that runs on the node via docker exec
func (n *Node) Cmder() exec.Cmder {
	if n.nodeCache.containerCmder == nil {
		n.nodeCache.containerCmder = docker.ContainerCmder(n.nameOrID)
	}
	return n.nodeCache.containerCmder
}

// Command returns a new exec.Cmd that will run on the node
func (n *Node) Command(command string, args ...string) exec.Cmd {
	return n.Cmder().Command(command, args...)
}

// this is a seperate struct so we can clearly the whole thing at once
// it contains lazily initialized fields
// like node.nodeCache = nodeCache{}
type nodeCache struct {
	kubernetesVersion string
	ip                string
	ports             map[int]int
	role              config.NodeRole
	containerCmder    exec.Cmder
}

func (n *Node) String() string {
	return n.nameOrID
}

// SignalStart sends SIGUSR1 to the node, which signals our entrypoint to boot
// see images/node/entrypoint
func (n *Node) SignalStart() error {
	return docker.Kill("SIGUSR1", n.nameOrID)
}

// CopyTo copies the source file on the host to dest on the node
func (n *Node) CopyTo(source, dest string) error {
	return docker.CopyTo(source, n.nameOrID, dest)
}

// CopyFrom copies the source file on the node to dest on the host
// TODO(fabrizio pandini): note that this does have limitations around symlinks
//     but this should go away when kubeadm automatic copy certs lands,
//     otherwise it should be refactored in something more robust in the long term
func (n *Node) CopyFrom(source, dest string) error {
	return docker.CopyFrom(n.nameOrID, source, dest)
}

// WaitForDocker waits for Docker to be ready on the node
// it returns true on success, and false on a timeout
func (n *Node) WaitForDocker(until time.Time) bool {
	return tryUntil(until, func() bool {
		cmd := n.Command("systemctl", "is-active", "docker")
		out, err := exec.CombinedOutputLines(cmd)
		if err != nil {
			return false
		}
		return len(out) == 1 && out[0] == "active"
	})
}

// helper that calls `try()`` in a loop until the deadline `until`
// has passed or `try()`returns true, returns wether try ever returned true
func tryUntil(until time.Time, try func() bool) bool {
	for until.After(time.Now()) {
		if try() {
			return true
		}
	}
	return false
}

// LoadImages loads image tarballs stored on the node into docker on the node
func (n *Node) LoadImages() {
	// load images cached on the node into docker
	if err := n.Command(
		"find",
		"/kind/images",
		"-name", "*.tar",
		"-exec", "docker", "load", "-i", "{}", ";",
	).Run(); err != nil {
		log.Warningf("Failed to preload docker images: %v", err)
		return
	}

	// if this fails, we don't care yet, but try to get the kubernetes verison
	// and see if we can skip retagging for amd64
	// if this fails, we can just assume some unknown version and re-tag
	// in a future release of kind, we can probably drop v1.11 support
	// and remove the logic below this comment entirely
	if rawVersion, err := n.KubeVersion(); err == nil {
		if ver, err := version.ParseGeneric(rawVersion); err == nil {
			if !ver.LessThan(version.MustParseSemantic("v1.12.0")) {
				return
			}
		}
	}

	// for older releases, we need the images to have the arch in their name
	// bazel built images were missing these, newer releases do not use them
	// for any builds ...
	// retag images that are missing -amd64 as image:tag -> image-amd64:tag
	// TODO(bentheelder): this is a bit gross, move this logic out of bash
	if err := n.Command(
		"/bin/bash", "-c",
		`docker images --format='{{.Repository}}:{{.Tag}}' | grep -v amd64 | xargs -L 1 -I '{}' /bin/bash -c 'docker tag "{}" "$(echo "{}" | sed s/:/-amd64:/)"'`,
	).Run(); err != nil {
		log.Warningf("Failed to re-tag docker images: %v", err)
	}
}

// FixMounts will correct mounts in the node container to meet the right
// sharing and permissions for systemd and Docker / Kubernetes
func (n *Node) FixMounts() error {
	// Check if userns-remap is enabled
	if docker.UsernsRemap() {
		// The binary /bin/mount should be owned by root:root in order to execute
		// the following mount commands
		if err := n.Command("chown", "root:root", "/bin/mount").Run(); err != nil {
			return err
		}
		// The binary /bin/mount should have the setuid bit
		if err := n.Command("chmod", "-s", "/bin/mount").Run(); err != nil {
			return err
		}
	}

	// systemd-in-a-container should have read only /sys
	// https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
	// however, we need other things from `docker run --privileged` ...
	// and this flag also happens to make /sys rw, amongst other things
	if err := n.Command("mount", "-o", "remount,ro", "/sys").Run(); err != nil {
		return err
	}
	// kubernetes needs shared mount propagation
	if err := n.Command("mount", "--make-shared", "/").Run(); err != nil {
		return err
	}
	if err := n.Command("mount", "--make-shared", "/run").Run(); err != nil {
		return err
	}
	if err := n.Command("mount", "--make-shared", "/var/lib/docker").Run(); err != nil {
		return err
	}
	return nil
}

// KubeVersion returns the Kubernetes version installed on the node
func (n *Node) KubeVersion() (version string, err error) {
	// use the cached version first
	if n.nodeCache.kubernetesVersion != "" {
		return n.nodeCache.kubernetesVersion, nil
	}
	// grab kubernetes version from the node image
	cmd := n.Command("cat", "/kind/version")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("file should only be one line, got %d lines", len(lines))
	}
	n.nodeCache.kubernetesVersion = lines[0]
	return n.nodeCache.kubernetesVersion, nil
}

// IP returns the IP address of the node
func (n *Node) IP() (ip string, err error) {
	// use the cached version first
	if n.nodeCache.ip != "" {
		return n.nodeCache.ip, nil
	}
	// retrive the IP address of the node using docker inspect
	lines, err := docker.Inspect(n.nameOrID, "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}")
	if err != nil {
		return "", errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("file should only be one line, got %d lines", len(lines))
	}
	n.nodeCache.ip = lines[0]
	return n.nodeCache.ip, nil
}

// Ports returns a specific port mapping for the node
// Node by convention use well known ports internally, while random port
// are used for making the `kind` cluster accessible from the host machine
func (n *Node) Ports(containerPort int) (hostPort int, err error) {
	// use the cached version first
	if hostPort, ok := n.nodeCache.ports[containerPort]; ok {
		return hostPort, nil
	}
	// retrive the specific port mapping using docker inspect
	lines, err := docker.Inspect(n.nameOrID, fmt.Sprintf("{{(index (index .NetworkSettings.Ports \"%d/tcp\") 0).HostPort}}", containerPort))
	if err != nil {
		return -1, errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return -1, errors.Errorf("file should only be one line, got %d lines", len(lines))
	}

	if n.nodeCache.ports == nil {
		n.nodeCache.ports = map[int]int{}
	}

	n.nodeCache.ports[containerPort], err = strconv.Atoi(lines[0])
	if err != nil {
		return -1, errors.Wrap(err, "failed to get file")
	}
	return n.nodeCache.ports[containerPort], nil
}

// Role returns the role of the node
func (n *Node) Role() (role config.NodeRole, err error) {
	// use the cached version first
	if n.nodeCache.role != "" {
		return n.nodeCache.role, nil
	}
	// retrive the role the node using docker inspect
	lines, err := docker.Inspect(n.nameOrID, fmt.Sprintf("{{index .Config.Labels %q}}", constants.ClusterRoleKey))
	if err != nil {
		return "", errors.Wrapf(err, "failed to get %q label", constants.ClusterRoleKey)
	}
	if len(lines) != 1 {
		return "", errors.Errorf("%q label should only be one line, got %d lines", constants.ClusterRoleKey, len(lines))
	}
	n.nodeCache.role = config.NodeRole(strings.Trim(lines[0], "'"))
	return n.nodeCache.role, nil
}

// matches kubeconfig server entry like:
//    server: https://172.17.0.2:6443
// which we rewrite to:
//    server: https://localhost:$PORT
var serverAddressRE = regexp.MustCompile(`^(\s+server:) https://.*:\d+$`)

// WriteKubeConfig writes a fixed KUBECONFIG to dest
// this should only be called on a control plane node
// While copyng to the host machine the control plane address
// is replaced with local host and the control plane port with
// a randomly generated port reserved during node creation.
func (n *Node) WriteKubeConfig(dest string, hostPort int) error {
	cmd := n.Command("cat", "/etc/kubernetes/admin.conf")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	// fix the config file, swapping out the server for the forwarded localhost:port
	var buff bytes.Buffer
	for _, line := range lines {
		match := serverAddressRE.FindStringSubmatch(line)
		if len(match) > 1 {
			line = fmt.Sprintf("%s https://localhost:%d", match[1], hostPort)
		}
		buff.WriteString(line)
		buff.WriteString("\n")
	}

	// create the directory to contain the KUBECONFIG file.
	// 0755 is taken from client-go's config handling logic: https://github.com/kubernetes/client-go/blob/5d107d4ebc00ee0ea606ad7e39fd6ce4b0d9bf9e/tools/clientcmd/loader.go#L412
	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create kubeconfig output directory")
	}

	return ioutil.WriteFile(dest, buff.Bytes(), 0600)
}
