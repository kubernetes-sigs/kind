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
	"sigs.k8s.io/kind/pkg/fs"

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
		// TODO: picking ports locally is less than ideal with remote docker
		// but this is supposed to be an implementation detail and NOT picking
		// them breaks host reboot ...
		// For now remote docker + multi control plane is not supported
		apiServerPort = 0              // replaced with random ports
		apiServerAddress = "127.0.0.1" // only the LB needs to be non-local
		if clusterIsIPv6(cfg) {
			apiServerAddress = "::1" // only the LB needs to be non-local
		}
		// plan loadbalancer node
		name := nodeNamer(constants.ExternalLoadBalancerNodeRoleValue)
		createContainerFuncs = append(createContainerFuncs, func() error {
			args, err := runArgsForLoadBalancer(cfg, name, genericArgs)
			if err != nil {
				return err
			}
			return createContainer(args)
		})
	}

	// plan normal nodes
	for _, node := range cfg.Nodes {
		node := node.DeepCopy()              // copy so we can modify
		name := nodeNamer(string(node.Role)) // name the node

		// fixup relative paths, docker can only handle absolute paths
		for i := range node.ExtraMounts {
			hostPath := node.ExtraMounts[i].HostPath
			if !fs.IsAbs(hostPath) {
				absHostPath, err := filepath.Abs(hostPath)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to resolve absolute path for hostPath: %q", hostPath)
				}
				node.ExtraMounts[i].HostPath = absHostPath
			}
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
				args, err := runArgsForNode(node, cfg.Networking.IPFamily, name, genericArgs)
				if err != nil {
					return err
				}
				return createContainer(args)
			})
		case config.WorkerRole:
			createContainerFuncs = append(createContainerFuncs, func() error {
				args, err := runArgsForNode(node, cfg.Networking.IPFamily, name, genericArgs)
				if err != nil {
					return err
				}
				return createContainer(args)
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
		// user a user defined docker network so we get embedded DNS
		"--net", fixedNetworkName,
		// Docker supports the following restart modes:
		// - no
		// - on-failure[:max-retries]
		// - unless-stopped
		// - always
		// https://docs.docker.com/engine/reference/commandline/run/#restart-policies---restart
		//
		// What we desire is:
		// - restart on host / dockerd reboot
		// - don't restart for any other reason
		//
		// This means:
		// - no is out of the question ... it never restarts
		// - always is a poor choice, we'll keep trying to restart nodes that were
		// never going to work
		// - unless-stopped will also retry failures indefinitely, similar to always
		// except that it won't restart when the container is `docker stop`ed
		// - on-failure is not great, we're only interested in restarting on
		// reboots, not failures. *however* we can limit the number of retries
		// *and* it forgets all state on dockerd restart and retries anyhow.
		// - on-failure:0 is what we want .. restart on failures, except max
		// retries is 0, so only restart on reboots.
		// however this _actually_ means the same thing as always
		// so the closest thing is on-failure:1, which will retry *once*
		"--restart=on-failure:1",
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

	// Nestybox: with Sysbox, KinD works inside containers using userns-remap.
	// if usernsRemap() {
	// 	args = append(args, "--userns=host")
	// }

	// handle Docker on Btrfs or ZFS
	// https://github.com/kubernetes-sigs/kind/issues/1416#issuecomment-606514724
	if mountDevMapper() {
		args = append(args, "--volume", "/dev/mapper:/dev/mapper")
	}

	return args, nil
}

func runArgsForNode(node *config.Node, clusterIPFamily config.ClusterIPFamily, name string, args []string) ([]string, error) {
	args = append([]string{
		"run",

		// Nestybox: use Sysbox
		"--runtime=sysbox-runc",

		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", nodeRoleLabelKey, node.Role),

		// Nestybox: with Sysbox, the following are not required.
		//"--privileged",
		//"--security-opt", "seccomp=unconfined", // also ignore seccomp
		//"--security-opt", "apparmor=unconfined", // also ignore apparmor
		//"--tmpfs", "/tmp", // various things depend on working /tmp
		//"--tmpfs", "/run", // systemd wants a writable /run
		//"--volume", "/var",
		//"--volume", "/lib/modules:/lib/modules:ro",
	},
		args...,
	)

	// convert mounts and port mappings to container run args
	args = append(args, generateMountBindings(node.ExtraMounts...)...)
	mappingArgs, err := generatePortMappings(clusterIPFamily, node.ExtraPortMappings...)
	if err != nil {
		return nil, err
	}
	args = append(args, mappingArgs...)

	// finally, specify the image to run
	return append(args, node.Image), nil
}

func runArgsForLoadBalancer(cfg *config.Cluster, name string, args []string) ([]string, error) {
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
	mappingArgs, err := generatePortMappings(cfg.Networking.IPFamily,
		config.PortMapping{
			ListenAddress: cfg.Networking.APIServerAddress,
			HostPort:      cfg.Networking.APIServerPort,
			ContainerPort: common.APIServerInternalPort,
		},
	)
	if err != nil {
		return nil, err
	}
	args = append(args, mappingArgs...)

	// finally, specify the image to run
	return append(args, loadbalancer.Image), nil
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
func generatePortMappings(clusterIPFamily config.ClusterIPFamily, portMappings ...config.PortMapping) ([]string, error) {
	args := make([]string, 0, len(portMappings))
	for _, pm := range portMappings {
		// do provider internal defaulting
		// in a future API revision we will handle this at the API level and remove this
		if pm.ListenAddress == "" {
			switch clusterIPFamily {
			case config.IPv4Family:
				pm.ListenAddress = "0.0.0.0" // this is the docker default anyhow
			case config.IPv6Family:
				pm.ListenAddress = "::"
			default:
				return nil, errors.Errorf("unknown cluster IP family: %v", clusterIPFamily)
			}
		}
		if string(pm.Protocol) == "" {
			pm.Protocol = config.PortMappingProtocolTCP // TCP is the default
		}

		// validate that the provider can handle this binding
		switch pm.Protocol {
		case config.PortMappingProtocolTCP:
		case config.PortMappingProtocolUDP:
		case config.PortMappingProtocolSCTP:
		default:
			return nil, errors.Errorf("unknown port mapping protocol: %v", pm.Protocol)
		}

		// get a random port if necessary (port = 0)
		hostPort, err := common.PortOrGetFreePort(pm.HostPort, pm.ListenAddress)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get random host port for port mapping")
		}

		// generate the actual mapping arg
		protocol := string(pm.Protocol)
		hostPortBinding := net.JoinHostPort(pm.ListenAddress, fmt.Sprintf("%d", hostPort))
		args = append(args, fmt.Sprintf("--publish=%s:%d/%s", hostPortBinding, pm.ContainerPort, protocol))
	}
	return args, nil
}
