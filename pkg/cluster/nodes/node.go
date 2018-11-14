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

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/exec"
)

// Node represents a handle to a kind node
// This struct must be created by one of: CreateControlPlane
// It should not be manually instantiated
type Node struct {
	// must be one of docker container ID or name
	nameOrID string
	// cached node info
	cachedNodeInfo
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

// Run execs command, args... on the node
func (n *Node) Run(command string, args ...string) error {
	args = append(
		[]string{
			"exec",
			"-t",           // use a tty so we can get output
			"--privileged", // run with priliges so we can remount etc..
			n.nameOrID,     // ... against the "node" container
			command,        // with the command specified
		},
		// finally, with the caller args
		args...,
	)
	cmd := exec.Command("docker", args...)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

// RunQ execs command, args... on the node without inherting stdout
func (n *Node) RunQ(command string, args ...string) error {
	args = append(
		[]string{
			"exec",
			"-t",           // use a tty so we can get output
			"--privileged", // run with priliges so we can remount etc..
			n.nameOrID,     // ... against the "node" container
			command,        // with the command specified
		},
		// finally, with the caller args
		args...,
	)
	cmd := exec.Command("docker", args...)
	return cmd.Run()
}

// RunWithInput execs command, args... on the node, hooking input to stdin
func (n *Node) RunWithInput(input io.Reader, command string, args ...string) error {
	args = append(
		[]string{
			"exec",
			"-t",           // use a tty so we can get output
			"--privileged", // run with priliges so we can remount etc..
			n.nameOrID,     // ... against the "node" container
			command,        // with the command specified
		},
		// finally, with the caller args
		args...,
	)
	cmd := exec.Command("docker", args...)
	exec.InheritOutput(cmd)
	cmd.SetStdin(input)
	return cmd.Run()
}

// RunQWithInput execs command, args... on the node, hooking input to stdin
func (n *Node) RunQWithInput(input io.Reader, command string, args ...string) error {
	args = append(
		[]string{
			"exec",
			"-i",           // interactive so we can supply input
			"--privileged", // run with priliges so we can remount etc..
			n.nameOrID,     // ... against the "node" container
			command,        // with the command specified
		},
		// finally, with the caller args
		args...,
	)
	cmd := exec.Command("docker", args...)
	cmd.SetStdin(input)
	return cmd.Run()
}

// RunHook runs a LifecycleHook on the node
// It will only return an error if hook.MustSucceed is true
func (n *Node) RunHook(hook *config.LifecycleHook, phase string) error {
	logger := log.WithFields(log.Fields{
		"node":  n.nameOrID,
		"phase": phase,
	})
	if hook.Name != "" {
		logger.Infof("Running LifecycleHook \"%s\" ...", hook.Name)
	} else {
		logger.Info("Running LifecycleHook ...")
	}
	if err := n.Run(hook.Command[0], hook.Command[1:]...); err != nil {
		if hook.MustSucceed {
			logger.WithError(err).Error("LifecycleHook failed")
			return err
		}
		logger.WithError(err).Warn("LifecycleHook failed, continuing ...")
	}
	return nil
}

// CombinedOutputLines execs command, args... on the node, returning the output lines
func (n *Node) CombinedOutputLines(command string, args ...string) ([]string, error) {
	args = append(
		[]string{
			"exec",
			"-t",           // use a tty so we can get output
			"--privileged", // run with priliges so we can remount etc..
			n.nameOrID,     // ... against the "node" container
			command,        // with the command specified
		},
		// finally, with the caller args
		args...,
	)
	cmd := exec.Command("docker", args...)
	return exec.CombinedOutputLines(cmd)
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
		out, err := n.CombinedOutputLines("systemctl", "is-active", "docker")
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
	if err := n.RunQ(
		"find",
		"/kind/images",
		"-name", "*.tar",
		"-exec", "docker", "load", "-i", "{}", ";",
	); err != nil {
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
	if err := n.RunQ(
		"/bin/bash", "-c",
		`docker images --format='{{.Repository}}:{{.Tag}}' | grep -v amd64 | xargs -L 1 -I '{}' /bin/bash -c 'docker tag "{}" "$(echo "{}" | sed s/:/-amd64:/)"'`,
	); err != nil {
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
	if err := n.RunQ("mount", "-o", "remount,ro", "/sys"); err != nil {
		return err
	}
	// kubernetes needs shared mount propagation
	if err := n.RunQ("mount", "--make-shared", "/"); err != nil {
		return err
	}
	if err := n.RunQ("mount", "--make-shared", "/run"); err != nil {
		return err
	}
	if err := n.RunQ("mount", "--make-shared", "/var/lib/docker"); err != nil {
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
	lines, err := n.CombinedOutputLines("cat", "/kind/version")
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
	lines, err := n.CombinedOutputLines("cat", "/etc/kubernetes/admin.conf")
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
