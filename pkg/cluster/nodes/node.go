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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/constants"

	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/exec"
)

const (
	// Docker default bridge network is named "bridge" (https://docs.docker.com/network/bridge/#use-the-default-bridge-network)
	defaultNetwork = "bridge"
	httpProxy      = "HTTP_PROXY"
	httpsProxy     = "HTTPS_PROXY"
	noProxy        = "NO_PROXY"
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
	ipv4              string
	ipv6              string
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

func (cache *nodeCache) IP() (string, string) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.ipv4, cache.ipv6
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
func (n *Node) IP() (ipv4 string, ipv6 string, err error) {
	// use the cached version first
	cachedIPv4, cachedIPv6 := n.cache.IP()
	// TODO: this assumes there are always ipv4 and ipv6 cached addresses
	if cachedIPv4 != "" && cachedIPv6 != "" {
		return cachedIPv4, cachedIPv6, nil
	}
	// retrive the IP address of the node using docker inspect
	lines, err := docker.Inspect(n.name, "{{range .NetworkSettings.Networks}}{{.IPAddress}},{{.GlobalIPv6Address}}{{end}}")
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get container details")
	}
	if len(lines) != 1 {
		return "", "", errors.Errorf("file should only be one line, got %d lines", len(lines))
	}
	ips := strings.Split(lines[0], ",")
	if len(ips) != 2 {
		return "", "", errors.Errorf("container addresses should have 2 values, got %d values", len(ips))
	}
	n.cache.set(func(cache *nodeCache) {
		cache.ipv4 = ips[0]
		cache.ipv6 = ips[1]
	})
	return ips[0], ips[1], nil
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

// WriteFile writes content to dest on the node
func (n *Node) WriteFile(dest, content string) error {
	// create destination directory
	cmd := n.Command("mkdir", "-p", filepath.Dir(dest))
	err := exec.RunLoggingOutputOnFail(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", dest)
	}

	return n.Command("cp", "/dev/stdin", dest).SetStdin(strings.NewReader(content)).Run()
}

// ImageInspect return low-level information on containers images inside a node
func (n *Node) ImageInspect(containerNameOrID string) ([]string, error) {
	cmd := n.Command(
		"crictl", "inspecti", containerNameOrID,
	)
	return exec.CombinedOutputLines(cmd)
}

// LoadImageArchive will load the image contents in the image reader to the
// k8s.io namespace on the node such that the image can be used from a
// Kubernetes pod
func (n *Node) LoadImageArchive(image io.Reader) error {
	cmd := n.Command(
		"ctr", "--namespace=k8s.io", "images", "import", "-",
	)
	cmd.SetStdin(image)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to load image")
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
func getProxyDetails() (*proxyDetails, error) {
	var proxyEnvs = []string{httpProxy, httpsProxy, noProxy}
	var val string
	var details proxyDetails
	details.Envs = make(map[string]string)

	proxySupport := false

	for _, name := range proxyEnvs {
		val = os.Getenv(name)
		if val != "" {
			proxySupport = true
			details.Envs[name] = val
			details.Envs[strings.ToLower(name)] = val
		} else {
			val = os.Getenv(strings.ToLower(name))
			if val != "" {
				proxySupport = true
				details.Envs[name] = val
				details.Envs[strings.ToLower(name)] = val
			}
		}
	}

	// Specifically add the docker network subnets to NO_PROXY if we are using proxies
	if proxySupport {
		subnets, err := getSubnets(defaultNetwork)
		if err != nil {
			return nil, err
		}
		noProxyList := strings.Join(append(subnets, details.Envs[noProxy]), ",")
		details.Envs[noProxy] = noProxyList
		details.Envs[strings.ToLower(noProxy)] = noProxyList
	}

	return &details, nil
}

// getSubnets returns a slice of subnets for a specified network
func getSubnets(networkName string) ([]string, error) {
	format := `{{range (index (index . "IPAM") "Config")}}{{index . "Subnet"}} {{end}}`
	lines, err := docker.NetworkInspect([]string{networkName}, format)
	if err != nil {
		return nil, err
	}
	return strings.Split(lines[0], " "), nil
}

// EnableIPv6 enables IPv6 inside the node container and in the inner docker daemon
func (n *Node) EnableIPv6() error {
	// enable ipv6
	cmd := n.Command("sysctl", "net.ipv6.conf.all.disable_ipv6=0")
	err := exec.RunLoggingOutputOnFail(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to enable ipv6")
	}
	// enable ipv6 forwarding
	cmd = n.Command("sysctl", "net.ipv6.conf.all.forwarding=1")
	err = exec.RunLoggingOutputOnFail(cmd)
	if err != nil {
		return errors.Wrap(err, "failed to enable ipv6 forwarding")
	}
	return nil
}
