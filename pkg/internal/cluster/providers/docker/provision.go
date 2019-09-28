/*
Copyright 2019 The Kubernetes Authors.

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

package docker

import (
	"fmt"
	"net"
	"os"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/container/cri"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/util/concurrent"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cluster/loadbalancer"
	"sigs.k8s.io/kind/pkg/internal/cluster/providers/provider/common"
)

// provision creates the docker node containers
func provision(cluster string, cfg *config.Cluster) error {
	// these are used by all node creation
	nodeNamer := common.MakeNodeNamer(cluster)
	clusterLabel := fmt.Sprintf("%s=%s", constants.ClusterLabelKey, cluster)

	// determine if we are HA, and what the LB will need to know for that
	controlPlanes := 0
	for _, configNode := range cfg.Nodes {
		role := string(configNode.Role)
		if role == constants.ControlPlaneNodeRoleValue {
			controlPlanes++
		}
	}
	isHA := controlPlanes > 1
	isIPv6 := cfg.Networking.IPFamily == "ipv6"

	// only the external LB should reflect the port if we have
	// multiple control planes
	apiServerPort := cfg.Networking.APIServerPort
	apiServerAddress := cfg.Networking.APIServerAddress
	if isHA {
		apiServerPort = 0              // replaced with a random port
		apiServerAddress = "127.0.0.1" // only the LB needs to be non-local
		if isIPv6 {
			apiServerAddress = "::1" // only the LB needs to be non-local
		}

	}

	// plan node creation
	createNodeFuncs := []func() error{}
	for _, configNode := range cfg.Nodes {
		configNode := configNode // capture loop variable
		role := string(configNode.Role)
		nodeName := nodeNamer(role)
		switch role {
		case constants.ControlPlaneNodeRoleValue:
			createNodeFuncs = append(createNodeFuncs, func() error {
				return createControlPlaneNode(nodeName, clusterLabel, apiServerAddress, apiServerPort, &configNode, isIPv6)
			})
		case constants.WorkerNodeRoleValue:
			createNodeFuncs = append(createNodeFuncs, func() error {
				return createWorkerNode(nodeName, clusterLabel, &configNode, isIPv6)
			})
		default:
			return errors.Errorf("unknown node role: %q", role)
		}
	}
	if isHA {
		name := nodeNamer(constants.ExternalLoadBalancerNodeRoleValue)
		createNodeFuncs = append(createNodeFuncs, func() error {
			return createExternalLoadBalancerNode(name, clusterLabel, cfg.Networking.APIServerAddress, cfg.Networking.APIServerPort, isIPv6)
		})
	}

	// create nodes
	return concurrent.UntilError(createNodeFuncs)
}

// createControlPlaneNode creates a control-plane node
// and gets ready for exposing the API server
func createControlPlaneNode(name, clusterLabel, listenAddress string, port int32, node *config.Node, isIPv6 bool) error {
	// gets a random host port for the API server
	if port == 0 {
		p, err := common.GetFreePort(listenAddress)
		if err != nil {
			return errors.Wrap(err, "failed to get port for API server")
		}
		port = p
	}

	// add api server port mapping
	portMappingsWithAPIServer := append(node.ExtraPortMappings, cri.PortMapping{
		ListenAddress: listenAddress,
		HostPort:      port,
		ContainerPort: common.APIServerInternalPort,
	})
	return createNodeHelper(
		name, node.Image, clusterLabel, constants.ControlPlaneNodeRoleValue, node.ExtraMounts, portMappingsWithAPIServer, isIPv6,
		// publish selected port for the API server
		"--expose", fmt.Sprintf("%d", port),
	)
}

// createWorkerNode creates a worker node
func createWorkerNode(name, clusterLabel string, node *config.Node, isIPv6 bool) error {
	return createNodeHelper(name, node.Image, clusterLabel, constants.WorkerNodeRoleValue, node.ExtraMounts, node.ExtraPortMappings, isIPv6)
}

// createExternalLoadBalancerNode creates an external load balancer node
// and gets ready for exposing the API server and the load balancer admin console
func createExternalLoadBalancerNode(name, clusterLabel, listenAddress string, port int32, isIPv6 bool) error {
	// gets a random host port for control-plane load balancer
	// gets a random host port for the API server
	if port == 0 {
		p, err := common.GetFreePort(listenAddress)
		if err != nil {
			return errors.Wrap(err, "failed to get port for API server")
		}
		port = p
	}

	// load balancer port mapping
	portMappings := []cri.PortMapping{{
		ListenAddress: listenAddress,
		HostPort:      port,
		ContainerPort: common.APIServerInternalPort,
	}}

	// TODO: should not use the same create node code
	return createNodeHelper(name, loadbalancer.Image, clusterLabel, constants.ExternalLoadBalancerNodeRoleValue,
		nil, portMappings, isIPv6,
		// publish selected port for the control plane
		"--expose", fmt.Sprintf("%d", port),
	)
}

// TODO(bentheelder): refactor this to not have extraArgs
// createNodeHelper `docker run`s the node image, note that due to
// images/node/entrypoint being the entrypoint, this container will
// effectively be paused until we call actuallyStartNode(...)
func createNodeHelper(name, image, clusterLabel, role string, mounts []cri.Mount, portMappings []cri.PortMapping, isIPv6 bool, extraArgs ...string) error {
	args := []string{
		"run",
		"--detach", // run the container detached
		"--tty",    // allocate a tty for entrypoint logs
		// running containers in a container requires privileged
		// NOTE: we could try to replicate this with --cap-add, and use less
		// privileges, but this flag also changes some mounts that are necessary
		// including some ones docker would otherwise do by default.
		// for now this is what we want. in the future we may revisit this.
		"--privileged",
		"--security-opt", "seccomp=unconfined", // also ignore seccomp
		// runtime temporary storage
		"--tmpfs", "/tmp", // various things depend on working /tmp
		"--tmpfs", "/run", // systemd wants a writable /run
		// runtime persistent storage
		// this ensures that E.G. pods, logs etc. are not on the container
		// filesystem, which is not only better for performance, but allows
		// running kind in kind for "party tricks"
		// (please don't depend on doing this though!)
		"--volume", "/var",
		// some k8s things want to read /lib/modules
		"--volume", "/lib/modules:/lib/modules:ro",
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the cluster ID
		"--label", clusterLabel,
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", constants.NodeRoleKey, role),
	}

	// pass proxy environment variables to be used by node's docker daemon
	proxyDetails, err := getProxyDetails()
	if err != nil || proxyDetails == nil {
		return errors.Wrap(err, "proxy setup error")
	}
	for key, val := range proxyDetails.Envs {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	if isIPv6 {
		args = append(args, "--sysctl", "net.ipv6.conf.all.disable_ipv6=0", "--sysctl", "net.ipv6.conf.all.forwarding=1")
	}

	// adds node specific args
	args = append(args, extraArgs...)

	if usernsRemap() {
		// We need this argument in order to make this command work
		// in systems that have userns-remap enabled on the docker daemon
		args = append(args, "--userns=host")
	}

	// convert mounts to container run args
	for _, mount := range mounts {
		bindings, err := generateMountBindings(mount)
		if err != nil {
			return err
		}
		args = append(args, bindings...)
	}
	for _, portMapping := range portMappings {
		args = append(args, generatePortMappings(portMapping)...)
	}

	// finally, specify the image to run
	args = append(args, image)

	// construct the actual docker run argv
	if err := exec.Command("docker", args...).Run(); err != nil {
		return errors.Wrap(err, "docker run error")
	}

	return nil
}

const (
	// Docker default bridge network is named "bridge" (https://docs.docker.com/network/bridge/#use-the-default-bridge-network)
	defaultNetwork = "bridge"
	httpProxy      = "HTTP_PROXY"
	httpsProxy     = "HTTPS_PROXY"
	noProxy        = "NO_PROXY"
)

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
	cmd := exec.Command("docker", "network", "inspect",
		"-f", format, networkName,
	)
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subnets")
	}
	return strings.Split(strings.TrimSpace(lines[0]), " "), nil
}

/*
This is adapated from:
https://github.com/kubernetes/kubernetes/blob/07a5488b2a8f67add543da72e8819407d8314204/pkg/kubelet/dockershim/helpers.go#L115-L155
*/
// generateMountBindings converts the mount list to a list of strings that
// can be understood by docker.
// '<HostPath>:<ContainerPath>[:options]', where 'options'
// is a comma-separated list of the following strings:
// 'ro', if the path is read only
// 'Z', if the volume requires SELinux relabeling
func generateMountBindings(mounts ...cri.Mount) ([]string, error) {
	result := make([]string, 0, len(mounts))
	for _, m := range mounts {
		bind := fmt.Sprintf("%s:%s", m.HostPath, m.ContainerPath)
		var attrs []string
		if m.Readonly {
			attrs = append(attrs, "ro")
		}
		// Only request relabeling if the pod provides an SELinux context. If the pod
		// does not provide an SELinux context relabeling will label the volume with
		// the container's randomly allocated MCS label. This would restrict access
		// to the volume to the container which mounts it first.
		if m.SelinuxRelabel {
			attrs = append(attrs, "Z")
		}
		switch m.Propagation {
		case cri.MountPropagationNone:
			// noop, private is default
		case cri.MountPropagationBidirectional:
			attrs = append(attrs, "rshared")
		case cri.MountPropagationHostToContainer:
			attrs = append(attrs, "rslave")
		default:
			return nil, errors.Errorf("unknown propagation mode for hostPath %q", m.HostPath)
			// Falls back to "private"
		}

		if len(attrs) > 0 {
			bind = fmt.Sprintf("%s:%s", bind, strings.Join(attrs, ","))
		}
		// our specific modification is the following line: make this a docker flag
		bind = fmt.Sprintf("--volume=%s", bind)
		result = append(result, bind)
	}
	return result, nil
}

func generatePortMappings(portMappings ...cri.PortMapping) []string {
	result := make([]string, 0, len(portMappings))
	for _, pm := range portMappings {
		var hostPortBinding string
		if pm.ListenAddress != "" {
			hostPortBinding = net.JoinHostPort(pm.ListenAddress, fmt.Sprintf("%d", pm.HostPort))
		} else {
			hostPortBinding = fmt.Sprintf("%d", pm.HostPort)
		}
		var protocol string
		switch pm.Protocol {
		case cri.PortMappingProtocolTCP:
			protocol = "TCP"
		case cri.PortMappingProtocolUDP:
			protocol = "UDP"
		case cri.PortMappingProtocolSCTP:
			protocol = "SCTP"
		default:
			protocol = "TCP"
		}
		publish := fmt.Sprintf("--publish=%s:%d/%s", hostPortBinding, pm.ContainerPort, protocol)
		result = append(result, publish)
	}
	return result
}
