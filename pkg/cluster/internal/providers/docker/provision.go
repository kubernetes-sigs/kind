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
	"path/filepath"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/provider/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// planCreation creates a slice of funcs that will create the containers
func planCreation(cluster string, cfg *config.Cluster) (createContainerFuncs []func() error, err error) {
	// these apply to all container creation
	nodeNamer := common.MakeNodeNamer(cluster)
	genericArgs, err := commonArgs(cluster, cfg)
	if err != nil {
		return nil, err
	}

	// only the external LB should reflect the port if we have multiple control planes
	apiServerPort := cfg.Networking.APIServerPort
	apiServerAddress := cfg.Networking.APIServerAddress
	if clusterHasImplicitLoadBalancer(cfg) {
		apiServerPort = 0              // replaced with random ports
		apiServerAddress = "127.0.0.1" // only the LB needs to be non-local
		if clusterIsIPv6(cfg) {
			apiServerAddress = "::1" // only the LB needs to be non-local
		}
		// plan loadbalancer node
		name := nodeNamer(constants.ExternalLoadBalancerNodeRoleValue)
		createContainerFuncs = append(createContainerFuncs, func() error {
			return createContainer(runArgsForLoadBalancer(cfg, name, genericArgs))
		})
	}

	// plan normal nodes
	for _, node := range cfg.Nodes {
		node := node.DeepCopy()              // copy so we can modify
		name := nodeNamer(string(node.Role)) // name the node

		// fixup relative paths, docker can only handle absolute paths
		for i := range node.ExtraMounts {
			hostPath := node.ExtraMounts[i].HostPath
			absHostPath, err := filepath.Abs(hostPath)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to resolve absolute path for hostPath: %q", hostPath)
			}
			node.ExtraMounts[i].HostPath = absHostPath
		}

		// plan actual creation based on role
		switch node.Role {
		case config.ControlPlaneRole:
			createContainerFuncs = append(createContainerFuncs, func() error {
				node.ExtraPortMappings = append(node.ExtraPortMappings,
					config.PortMapping{
						ListenAddress: apiServerAddress,
						HostPort:      apiServerPort,
						ContainerPort: common.APIServerInternalPort,
					},
				)
				return createContainer(runArgsForNode(node, name, genericArgs))
			})
		case config.WorkerRole:
			createContainerFuncs = append(createContainerFuncs, func() error {
				return createContainer(runArgsForNode(node, name, genericArgs))
			})
		default:
			return nil, errors.Errorf("unknown node role: %q", node.Role)
		}
	}
	return createContainerFuncs, nil
}

func createContainer(args []string) error {
	if err := exec.Command("docker", args...).Run(); err != nil {
		return errors.Wrap(err, "docker run error")
	}
	return nil
}

func clusterIsIPv6(cfg *config.Cluster) bool {
	return cfg.Networking.IPFamily == "ipv6"
}

func clusterHasImplicitLoadBalancer(cfg *config.Cluster) bool {
	controlPlanes := 0
	for _, configNode := range cfg.Nodes {
		role := string(configNode.Role)
		if role == constants.ControlPlaneNodeRoleValue {
			controlPlanes++
		}
	}
	return controlPlanes > 1
}

// commonArgs computes static arguments that apply to all containers
func commonArgs(cluster string, cfg *config.Cluster) ([]string, error) {
	// standard arguments all nodes containers need, computed once
	args := []string{
		"--detach", // run the container detached
		"--tty",    // allocate a tty for entrypoint logs
		// label the node with the cluster ID
		"--label", fmt.Sprintf("%s=%s", clusterLabelKey, cluster),
	}

	// enable IPv6 if necessary
	if clusterIsIPv6(cfg) {
		args = append(args, "--sysctl=net.ipv6.conf.all.disable_ipv6=0", "--sysctl=net.ipv6.conf.all.forwarding=1")
	}

	// pass proxy environment variables
	proxyEnv, err := getProxyEnv(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "proxy setup error")
	}
	for key, val := range proxyEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	// handle hosts that have user namespace remapping enabled
	if usernsRemap() {
		args = append(args, "--userns=host")
	}
	return args, nil
}

func runArgsForNode(node *config.Node, name string, args []string) []string {
	args = append([]string{
		"run",
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", nodeRoleLabelKey, node.Role),
		// running containers in a container requires privileged
		// NOTE: we could try to replicate this with --cap-add, and use less
		// privileges, but this flag also changes some mounts that are necessary
		// including some ones docker would otherwise do by default.
		// for now this is what we want. in the future we may revisit this.
		"--privileged",
		"--security-opt", "seccomp=unconfined", // also ignore seccomp
		"--security-opt", "apparmor=unconfined", // also ignore apparmor
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
	},
		args...,
	)

	// convert mounts and port mappings to container run args
	args = append(args, generateMountBindings(node.ExtraMounts...)...)
	args = append(args, generatePortMappings(node.ExtraPortMappings...)...)

	// finally, specify the image to run
	return append(args, node.Image)
}

func runArgsForLoadBalancer(cfg *config.Cluster, name string, args []string) []string {
	args = append([]string{
		"run",
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", nodeRoleLabelKey, constants.ExternalLoadBalancerNodeRoleValue),
	},
		args...,
	)

	// load balancer port mapping
	args = append(args, generatePortMappings(config.PortMapping{
		ListenAddress: cfg.Networking.APIServerAddress,
		HostPort:      cfg.Networking.APIServerPort,
		ContainerPort: common.APIServerInternalPort,
	})...)

	// finally, specify the image to run
	return append(args, loadbalancer.Image)
}

func getProxyEnv(cfg *config.Cluster) (map[string]string, error) {
	envs := common.GetProxyEnvs(cfg)
	// Specifically add the docker network subnets to NO_PROXY if we are using a proxy
	if len(envs) > 0 {
		// Docker default bridge network is named "bridge" (https://docs.docker.com/network/bridge/#use-the-default-bridge-network)
		subnets, err := getSubnets("bridge")
		if err != nil {
			return nil, err
		}
		noProxyList := strings.Join(append(subnets, envs[common.NOProxy]), ",")
		envs[common.NOProxy] = noProxyList
		envs[strings.ToLower(common.NOProxy)] = noProxyList
	}
	return envs, nil
}

func getSubnets(networkName string) ([]string, error) {
	format := `{{range (index (index . "IPAM") "Config")}}{{index . "Subnet"}} {{end}}`
	cmd := exec.Command("docker", "network", "inspect", "-f", format, networkName)
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subnets")
	}
	return strings.Split(strings.TrimSpace(lines[0]), " "), nil
}

// generateMountBindings converts the mount list to a list of args for docker
// '<HostPath>:<ContainerPath>[:options]', where 'options'
// is a comma-separated list of the following strings:
// 'ro', if the path is read only
// 'Z', if the volume requires SELinux relabeling
func generateMountBindings(mounts ...config.Mount) []string {
	args := make([]string, 0, len(mounts))
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
		case config.MountPropagationNone:
			// noop, private is default
		case config.MountPropagationBidirectional:
			attrs = append(attrs, "rshared")
		case config.MountPropagationHostToContainer:
			attrs = append(attrs, "rslave")
		default: // Falls back to "private"
		}
		if len(attrs) > 0 {
			bind = fmt.Sprintf("%s:%s", bind, strings.Join(attrs, ","))
		}
		args = append(args, fmt.Sprintf("--volume=%s", bind))
	}
	return args
}

// generatePortMappings converts the portMappings list to a list of args for docker
func generatePortMappings(portMappings ...config.PortMapping) []string {
	args := make([]string, 0, len(portMappings))
	for _, pm := range portMappings {
		var hostPortBinding string
		if pm.ListenAddress != "" {
			hostPortBinding = net.JoinHostPort(pm.ListenAddress, fmt.Sprintf("%d", pm.HostPort))
		} else {
			hostPortBinding = fmt.Sprintf("%d", pm.HostPort)
		}
		protocol := "TCP" // TCP is the default
		switch pm.Protocol {
		case config.PortMappingProtocolUDP:
			protocol = "UDP"
		case config.PortMappingProtocolSCTP:
			protocol = "SCTP"
		default: // also covers cri.PortMappingProtocolTCP
		}
		args = append(args, fmt.Sprintf("--publish=%s:%d/%s", hostPortBinding, pm.ContainerPort, protocol))
	}
	return args
}
