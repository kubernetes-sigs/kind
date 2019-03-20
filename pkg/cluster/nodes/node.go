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
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/kind/pkg/cluster/constants"

	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/util"
)

// Node represents a handle to a kind node
// This struct must be created by one of: CreateControlPlane
// It should not be manually instantiated
// Node impleemnts exec.Cmder
type Node struct {
	// must be one of docker container ID or name
	name string
	// cached node info etc.
	cache *nodeCache
}

// assert Node implements Cmder
var _ exec.Cmder = &Node{}

// Cmder returns an exec.Cmder that runs on the node via docker exec
func (n *Node) Cmder() exec.Cmder {
	return docker.ContainerCmder(n.name)
}

// Command returns a new exec.Cmd that will run on the node
func (n *Node) Command(command string, args ...string) exec.Cmd {
	return n.Cmder().Command(command, args...)
}

// this is a separate struct so we can more easily ensure that this portion is
// thread safe
type nodeCache struct {
	mu                sync.RWMutex
	kubernetesVersion string
	ip                string
	ports             map[int32]int32
	role              string
}

func (cache *nodeCache) set(setter func(*nodeCache)) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	setter(cache)
}

func (cache *nodeCache) KubeVersion() string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.kubernetesVersion
}

func (cache *nodeCache) IP() string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.ip
}

func (cache *nodeCache) HostPort(p int32) (int32, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	if cache.ports == nil {
		return 0, false
	}
	v, ok := cache.ports[p]
	return v, ok
}

func (cache *nodeCache) Role() string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.role
}

func (n *Node) String() string {
	return n.name
}

// Name returns the node's name
func (n *Node) Name() string {
	return n.name
}

// SignalStart sends SIGUSR1 to the node, which signals our entrypoint to boot
// see images/node/entrypoint
func (n *Node) SignalStart() error {
	return docker.Kill("SIGUSR1", n.name)
}

// CopyTo copies the source file on the host to dest on the node
func (n *Node) CopyTo(source, dest string) error {
	return docker.CopyTo(source, n.name, dest)
}

// CopyFrom copies the source file on the node to dest on the host
// TODO(fabrizio pandini): note that this does have limitations around symlinks
//     but this should go away when kubeadm automatic copy certs lands,
//     otherwise it should be refactored in something more robust in the long term
func (n *Node) CopyFrom(source, dest string) error {
	return docker.CopyFrom(n.name, source, dest)
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
		"/bin/bash", "-c",
		// use xargs to load images in parallel
		`find /kind/images -name *.tar -print0 | xargs -0 -n 1 -P $(nproc) docker load -i`,
	).Run(); err != nil {
		log.Warningf("Failed to preload docker images: %v", err)
		return
	}

	// if this fails, we don't care yet, but try to get the kubernetes version
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
		fmt.Sprintf(`docker images --format='{{.Repository}}:{{.Tag}}' | grep -v %s | xargs -L 1 -I '{}' /bin/bash -c 'docker tag "{}" "$(echo "{}" | sed s/:/-%s:/)"'`,
			util.GetArch(), util.GetArch()),
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
	cachedVersion := n.cache.KubeVersion()
	if cachedVersion != "" {
		return cachedVersion, nil
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
	version = lines[0]
	n.cache.set(func(cache *nodeCache) {
		cache.kubernetesVersion = version
	})
	return version, nil
}

// IP returns the IP address of the node
func (n *Node) IP() (ip string, err error) {
	// use the cached version first
	cachedIP := n.cache.IP()
	if cachedIP != "" {
		return cachedIP, nil
	}
	// retrive the IP address of the node using docker inspect
	lines, err := docker.Inspect(n.name, "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}")
	if err != nil {
		return "", errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("file should only be one line, got %d lines", len(lines))
	}
	ip = lines[0]
	n.cache.set(func(cache *nodeCache) {
		cache.ip = ip
	})
	return ip, nil
}

// Ports returns a specific port mapping for the node
// Node by convention use well known ports internally, while random port
// are used for making the `kind` cluster accessible from the host machine
func (n *Node) Ports(containerPort int32) (hostPort int32, err error) {
	// use the cached version first
	hostPort, isCached := n.cache.HostPort(containerPort)
	if isCached {
		return hostPort, nil
	}
	// retrive the specific port mapping using docker inspect
	lines, err := docker.Inspect(n.name, fmt.Sprintf("{{(index (index .NetworkSettings.Ports \"%d/tcp\") 0).HostPort}}", containerPort))
	if err != nil {
		return -1, errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return -1, errors.Errorf("file should only be one line, got %d lines", len(lines))
	}
	parsed, err := strconv.ParseInt(lines[0], 10, 32)
	if err != nil {
		return -1, errors.Wrap(err, "failed to get file")
	}
	hostPort = int32(parsed)
	// cache it
	n.cache.set(func(cache *nodeCache) {
		if cache.ports == nil {
			cache.ports = map[int32]int32{}
		}
		cache.ports[containerPort] = hostPort
	})
	return hostPort, nil
}

// Role returns the role of the node
func (n *Node) Role() (role string, err error) {
	role = n.cache.Role()
	// use the cached version first
	if role != "" {
		return role, nil
	}
	// retrive the role the node using docker inspect
	lines, err := docker.Inspect(n.name, fmt.Sprintf("{{index .Config.Labels %q}}", constants.NodeRoleKey))
	if err != nil {
		return "", errors.Wrapf(err, "failed to get %q label", constants.NodeRoleKey)
	}
	if len(lines) != 1 {
		return "", errors.Errorf("%q label should only be one line, got %d lines", constants.NodeRoleKey, len(lines))
	}
	role = strings.Trim(lines[0], "'")
	n.cache.set(func(cache *nodeCache) {
		cache.role = role
	})
	return role, nil
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
func (n *Node) WriteKubeConfig(dest string, hostPort int32) error {
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

// WriteFile writes temporary file on the host
// and copies it to the node
func (n *Node) WriteFile(dest, content string) error {
	// create destination directory
	cmd := n.Command("mkdir", "-p", filepath.Dir(dest))
	err := exec.RunLoggingOutputOnFail(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", dest)
	}

	// create temporary file
	ftmp, err := ioutil.TempFile("", "")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary file")
	}

	path := ftmp.Name()
	defer os.Remove(path)

	// write content to the temp file
	_, err = ftmp.WriteString(content)
	if err != nil {
		return errors.Wrap(err, "failed to write content to the file")
	}

	// copy the temp file from the host to the node
	if err := n.CopyTo(path, dest); err != nil {
		return errors.Wrap(err, "failed to copy file to the node")
	}
	return nil
}

// NeedProxy returns true if the host environment appears to have proxy settings
func NeedProxy() bool {
	details := getProxyDetails()
	return len(details.Envs) > 0
}

// SetProxy configures proxy settings for the node
//
// Currently it only creates systemd drop-in for Docker daemon
// as described in Docker documentation: https://docs.docker.com/config/daemon/systemd/#http-proxy
//
// See also: NeedProxy and getProxyDetails
func (n *Node) SetProxy() error {
	details := getProxyDetails()
	// configure Docker daemon to use proxy
	proxies := ""
	for key, val := range details.Envs {
		proxies += fmt.Sprintf("\"%s=%s\" ", key, val)
	}

	err := n.WriteFile("/etc/systemd/system/docker.service.d/http-proxy.conf",
		"[Service]\nEnvironment="+proxies)
	if err != nil {
		errors.Wrap(err, "failed to create http-proxy drop-in")
	}

	return nil
}

// proxyDetails contains proxy settings discovered on the host
type proxyDetails struct {
	Envs map[string]string
	// future proxy details here
}

// getProxyDetails returns a struct with the host environment proxy settings
// that should be passed to the nodes
func getProxyDetails() proxyDetails {
	var proxyEnvs = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}
	var val string
	var details proxyDetails
	details.Envs = make(map[string]string)

	for _, name := range proxyEnvs {
		val = os.Getenv(name)
		if val != "" {
			details.Envs[name] = val
		} else {
			val = os.Getenv(strings.ToLower(name))
			if val != "" {
				details.Envs[name] = val
			}
		}
	}
	return details
}
