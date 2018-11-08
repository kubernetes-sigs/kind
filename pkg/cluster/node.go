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
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/version"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/docker"
	"sigs.k8s.io/kind/pkg/exec"
)

type nodeHandle struct {
	// must be one of docker container ID or name
	nameOrID string
	// cached node info
	cachedNodeInfo
}

// this is a seperate struct so we can clear the whole thing at once
type cachedNodeInfo struct {
	kubernetesVersion string
}

func getPort() (int, error) {
	// get a free TCP port for the API server
	dummyListener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer dummyListener.Close()
	port := dummyListener.Addr().(*net.TCPAddr).Port
	return port, nil
}

// createControlPlaneNode `docker run`s the node image, note that due to
// images/node/entrypoint being the entrypoint, this container will
// effectively be paused until we call actuallyStartNode(...)
func createControlPlaneNode(name, image, clusterLabel string) (handle *nodeHandle, port int, err error) {
	port, err = getPort()
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to get port for API server")
	}
	id, err := docker.Run(
		image,
		[]string{
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
		},
		[]string{
			// explicitly pass the entrypoint argument
			"/sbin/init",
		},
	)
	// if there is a returned ID then we did create a container
	// we should return a handle so the caller can clean it up
	// we'll return a handle with the nice name though
	if id != "" {
		handle = &nodeHandle{
			nameOrID: name,
		}
	}
	if err != nil {
		return handle, 0, errors.Wrap(err, "docker run error")
	}
	return handle, port, nil
}

// SignalStart sends SIGUSR1 to the node, which signals our entrypoint to boot
// see images/node/entrypoint
func (nh *nodeHandle) SignalStart() error {
	cmd := exec.Command("docker", "kill")
	cmd.Args = append(cmd.Args,
		"-s", "SIGUSR1",
		nh.nameOrID,
	)
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

// RunQ execs command, args... on the node without inherting stdout
func (nh *nodeHandle) RunQ(command string, args ...string) error {
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
	return cmd.Run()
}

// RunWithInput execs command, args... on the node, hooking input to stdin
func (nh *nodeHandle) RunWithInput(input io.Reader, command string, args ...string) error {
	cmd := exec.Command("docker", "exec")
	cmd.Args = append(cmd.Args,
		"-i",           // interactive so we can supply input
		"--privileged", // run with priliges so we can remount etc..
		nh.nameOrID,    // ... against the "node" container
		command,        // with the command specified
	)
	cmd.Args = append(cmd.Args,
		args..., // finally, with the args specified
	)
	cmd.Stdin = input
	cmd.InheritOutput = true
	return cmd.Run()
}

// RunQWithInput execs command, args... on the node, hooking input to stdin
func (nh *nodeHandle) RunQWithInput(input io.Reader, command string, args ...string) error {
	cmd := exec.Command("docker", "exec")
	cmd.Args = append(cmd.Args,
		"-i",           // interactive so we can supply input
		"--privileged", // run with priliges so we can remount etc..
		nh.nameOrID,    // ... against the "node" container
		command,        // with the command specified
	)
	cmd.Args = append(cmd.Args,
		args..., // finally, with the args specified
	)
	cmd.Stdin = input
	return cmd.Run()
}

// RunHook runs a LifecycleHook on the node
// It will only return an error if hook.MustSucceed is true
func (nh *nodeHandle) RunHook(hook *config.LifecycleHook, phase string) error {
	logger := log.WithFields(log.Fields{
		"node":  nh.nameOrID,
		"phase": phase,
	})
	if hook.Name != "" {
		logger.Infof("Running LifecycleHook \"%s\" ...", hook.Name)
	} else {
		logger.Info("Running LifecycleHook ...")
	}
	if err := nh.Run(hook.Command[0], hook.Command[1:]...); err != nil {
		if hook.MustSucceed {
			logger.WithError(err).Error("LifecycleHook failed")
			return err
		}
		logger.WithError(err).Warn("LifecycleHook failed, continuing ...")
	}
	return nil
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
		source,               // from the source file
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
	for until.After(time.Now()) {
		if try() {
			return true
		}
	}
	return false
}

// LoadImages loads image tarballs stored on the node into docker on the node
func (nh *nodeHandle) LoadImages() {
	// load images cached on the node into docker
	if err := nh.RunQ(
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
	// if this fails, we can just assume some unkown version and re-tag
	// in a future release of kind, we can probably drop v1.11 support
	// and remove the logic below this comment entirely
	if rawVersion, err := nh.KubeVersion(); err == nil {
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
	if err := nh.RunQ(
		"/bin/bash", "-c",
		`docker images --format='{{.Repository}}:{{.Tag}}' | grep -v amd64 | xargs -L 1 -I '{}' /bin/bash -c 'docker tag "{}" "$(echo "{}" | sed s/:/-amd64:/)"'`,
	); err != nil {
		log.Warningf("Failed to re-tag docker images: %v", err)
	}
}

// FixMounts will correct mounts in the node container to meet the right
// sharing and permissions for systemd and Docker / Kubernetes
func (nh *nodeHandle) FixMounts() error {
	// systemd-in-a-container should have read only /sys
	// https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
	// however, we need other things from `docker run --privileged` ...
	// and this flag also happens to make /sys rw, amongst other things
	if err := nh.RunQ("mount", "-o", "remount,ro", "/sys"); err != nil {
		return err
	}
	// kubernetes needs shared mount propagation
	if err := nh.RunQ("mount", "--make-shared", "/"); err != nil {
		return err
	}
	if err := nh.RunQ("mount", "--make-shared", "/run"); err != nil {
		return err
	}
	if err := nh.RunQ("mount", "--make-shared", "/var/lib/docker"); err != nil {
		return err
	}
	return nil
}

// KubeVersion returns the Kubernetes version installed on the node
func (nh *nodeHandle) KubeVersion() (version string, err error) {
	// use the cached version first
	if nh.cachedNodeInfo.kubernetesVersion != "" {
		return nh.cachedNodeInfo.kubernetesVersion, nil
	}
	// grab kubernetes version from the node image
	lines, err := nh.CombinedOutputLines("cat", "/kind/version")
	if err != nil {
		return "", errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return "", fmt.Errorf("file should only be one line, got %d lines", len(lines))
	}
	nh.cachedNodeInfo.kubernetesVersion = lines[0]
	return nh.cachedNodeInfo.kubernetesVersion, nil
}

// matches kubeconfig server entry like:
//    server: https://172.17.0.2:6443
// which we rewrite to:
//    server: https://localhost:$PORT
var serverAddressRE = regexp.MustCompile(`^(\s+server:) https://.*:(\d+)$`)

// WriteKubeConfig writes a fixed KUBECONFIG to dest
// this should only be called on a control plane node
func (nh *nodeHandle) WriteKubeConfig(dest string) error {
	lines, err := nh.CombinedOutputLines("cat", "/etc/kubernetes/admin.conf")
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
