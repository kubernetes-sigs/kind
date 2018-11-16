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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/version"

	"sigs.k8s.io/kind/pkg/exec"
)

// Node represents a handle to a kind node
// This struct must be created by one of: CreateControlPlane
// It should not be manually instantiated
// Node impleemnts exec.Cmder
type Node struct {
	// must be one of docker container ID or name
	nameOrID string
	// cached node info
	cachedNodeInfo
}

// assert Node implements Cmder
var _ exec.Cmder = &Node{}

// Command returns a new exec.Cmd that will run on the node
func (n *Node) Command(command string, args ...string) exec.Cmd {
	return &nodeCmd{
		node:    n,
		command: command,
		args:    args,
	}
}

// nodeCmd implements exec.Cmd
type nodeCmd struct {
	node    *Node
	command string
	args    []string
	env     []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func (c *nodeCmd) Run() error {
	args := []string{
		"exec",
		"--privileged", // run with priliges so we can remount etc..
	}
	if c.stdin != nil {
		args = append(args,
			"-i", // interactive so we can supply input
		)
	}
	if c.stderr != nil || c.stdout != nil {
		args = append(args,
			"-t", // use a tty so we can get output
		)
	}
	// set env
	for _, env := range c.env {
		args = append(args, "-e", env)
	}
	// specify the container and command, after this everything will be
	// args the the command in the container rather than to docker
	args = append(
		args,
		c.node.nameOrID, // ... against the "node" container
		c.command,       // with the command specified
	)
	args = append(
		args,
		// finally, with the caller args
		c.args...,
	)
	cmd := exec.Command("docker", args...)
	if c.stdin != nil {
		cmd.SetStdin(c.stdin)
	}
	if c.stderr != nil {
		cmd.SetStderr(c.stderr)
	}
	if c.stdout != nil {
		cmd.SetStdout(c.stdout)
	}
	return cmd.Run()
}

func (c *nodeCmd) SetEnv(env ...string) {
	c.env = env
}

func (c *nodeCmd) SetStdin(r io.Reader) {
	c.stdin = r
}

func (c *nodeCmd) SetStdout(w io.Writer) {
	c.stdout = w
}

func (c *nodeCmd) SetStderr(w io.Writer) {
	c.stderr = w
}

func (n *Node) String() string {
	return n.nameOrID
}

// this is a seperate struct so we can clear the whole thing at once
type cachedNodeInfo struct {
	kubernetesVersion string
}

// SignalStart sends SIGUSR1 to the node, which signals our entrypoint to boot
// see images/node/entrypoint
func (n *Node) SignalStart() error {
	cmd := exec.Command(
		"docker", "kill",
		"-s", "SIGUSR1",
		n.nameOrID,
	)
	return cmd.Run()
}

// CopyTo copies the source file on the host to dest on the node
func (n *Node) CopyTo(source, dest string) error {
	cmd := exec.Command(
		"docker", "cp",
		source,              // from the source file
		n.nameOrID+":"+dest, // to the node, at dest
	)
	exec.InheritOutput(cmd)
	return cmd.Run()
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
	if n.cachedNodeInfo.kubernetesVersion != "" {
		return n.cachedNodeInfo.kubernetesVersion, nil
	}
	// grab kubernetes version from the node image
	cmd := n.Command("cat", "/kind/version")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return "", fmt.Errorf("file should only be one line, got %d lines", len(lines))
	}
	n.cachedNodeInfo.kubernetesVersion = lines[0]
	return n.cachedNodeInfo.kubernetesVersion, nil
}

// matches kubeconfig server entry like:
//    server: https://172.17.0.2:6443
// which we rewrite to:
//    server: https://localhost:$PORT
var serverAddressRE = regexp.MustCompile(`^(\s+server:) https://.*:(\d+)$`)

// WriteKubeConfig writes a fixed KUBECONFIG to dest
// this should only be called on a control plane node
func (n *Node) WriteKubeConfig(dest string) error {
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
			line = fmt.Sprintf("%s https://localhost:%s", match[1], match[len(match)-1])
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
