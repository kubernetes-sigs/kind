/*
Copyright 2026 The Kubernetes Authors.

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

package swarm

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// planCreation creates a slice of funcs that will create the containers,
// distributing them across the supplied hosts (round-robin, control-plane
// always on hosts[0]).
func planCreation(cfg *config.Cluster, networkName string, hosts []Host) (createContainerFuncs []func() error, err error) {
	if len(hosts) == 0 {
		return nil, errors.New("planCreation: no hosts")
	}
	nodeNamer := common.MakeNodeNamer(cfg.Name)
	names := make([]string, len(cfg.Nodes))
	for i, n := range cfg.Nodes {
		names[i] = nodeNamer(string(n.Role))
	}
	// the swarm provider does not support the in-cluster HA loadbalancer
	// (multiple control planes); single control plane is enforced upstream.

	// build a context -> Host index for explicit pinning via node.Host
	byCtx := make(map[string]Host, len(hosts))
	for _, h := range hosts {
		byCtx[h.Context] = h
	}

	// assign each node to a host:
	//   - if node.Host is set and matches a configured context, honor it
	//   - else control plane lands on hosts[0]
	//   - else workers round-robin starting from hosts[1]
	nodeHosts := make([]Host, len(cfg.Nodes))
	workerIdx := 0
	for i, n := range cfg.Nodes {
		if n.Host != "" {
			h, ok := byCtx[n.Host]
			if !ok {
				return nil, errors.Errorf("node %d: host %q is not in --hosts", i, n.Host)
			}
			nodeHosts[i] = h
			if n.Role != config.ControlPlaneRole {
				workerIdx++
			}
			continue
		}
		if n.Role == config.ControlPlaneRole {
			nodeHosts[i] = hosts[0]
			continue
		}
		nodeHosts[i] = hosts[(1+workerIdx)%len(hosts)]
		workerIdx++
	}

	apiServerPort := cfg.Networking.APIServerPort
	apiServerAddress := cfg.Networking.APIServerAddress

	for i, n := range cfg.Nodes {
		node := n.DeepCopy()
		name := names[i]
		host := nodeHosts[i]

		genericArgs, err := commonArgs(cfg.Name, cfg, networkName, host)
		if err != nil {
			return nil, err
		}

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
				args, err := runArgsForNode(node, cfg.Networking.IPFamily, name, host, genericArgs)
				if err != nil {
					return err
				}
				return createContainerWithWaitUntilSystemdReachesMultiUserSystem(host.Context, name, args)
			})
		case config.WorkerRole:
			createContainerFuncs = append(createContainerFuncs, func() error {
				args, err := runArgsForNode(node, cfg.Networking.IPFamily, name, host, genericArgs)
				if err != nil {
					return err
				}
				return createContainerWithWaitUntilSystemdReachesMultiUserSystem(host.Context, name, args)
			})
		default:
			return nil, errors.Errorf("unknown node role: %q", node.Role)
		}
	}
	if config.ClusterHasImplicitLoadBalancer(cfg) {
		return nil, errors.New("swarm provider does not support multiple control planes / implicit loadbalancer")
	}
	return createContainerFuncs, nil
}

// commonArgs computes the static docker-run arguments that apply to every
// node container on the given host.
func commonArgs(cluster string, cfg *config.Cluster, networkName string, host Host) ([]string, error) {
	args := []string{
		"--detach",
		"--tty",
		"--label", fmt.Sprintf("%s=%s", clusterLabelKey, cluster),
		"--label", fmt.Sprintf("%s=%s", hostLabelKey, host.Context),
		"--net", networkName,
		"--restart=on-failure:1",
		"--init=false",
		"--cgroupns=private",
	}

	if config.ClusterHasIPv6(cfg) {
		args = append(args, "--sysctl=net.ipv6.conf.all.disable_ipv6=0", "--sysctl=net.ipv6.conf.all.forwarding=1")
	}

	proxyEnv := common.GetProxyEnvs(cfg)
	for key, val := range proxyEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	if usernsRemap(host.Context) {
		args = append(args, "--userns=host")
	}

	if cfg.Networking.DNSSearch != nil {
		args = append(args, "-e", "KIND_DNS_SEARCH="+strings.Join(*cfg.Networking.DNSSearch, " "))
	}

	return args, nil
}

func runArgsForNode(node *config.Node, clusterIPFamily config.ClusterIPFamily, name string, host Host, args []string) ([]string, error) {
	args = append([]string{
		"--hostname", name,
		"--label", fmt.Sprintf("%s=%s", nodeRoleLabelKey, node.Role),
		"--privileged",
		"--security-opt", "seccomp=unconfined",
		"--security-opt", "apparmor=unconfined",
		"--tmpfs", "/tmp",
		"--tmpfs", "/run",
		"--volume", "/var",
		"--volume", "/lib/modules:/lib/modules:ro",
		"-e", "KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER",
	},
		args...,
	)

	args = append(args, generateMountBindings(node.ExtraMounts...)...)
	mappingArgs, err := generatePortMappings(clusterIPFamily, node.ExtraPortMappings...)
	if err != nil {
		return nil, err
	}
	args = append(args, mappingArgs...)

	if node.Role == config.ControlPlaneRole {
		args = append(args, "-e", "KUBECONFIG=/etc/kubernetes/admin.conf")
	}

	return append(args, node.Image), nil
}

func generateMountBindings(mounts ...config.Mount) []string {
	args := make([]string, 0, len(mounts))
	for _, m := range mounts {
		bind := fmt.Sprintf("%s:%s", m.HostPath, m.ContainerPath)
		var attrs []string
		if m.Readonly {
			attrs = append(attrs, "ro")
		}
		if m.SelinuxRelabel {
			attrs = append(attrs, "Z")
		}
		switch m.Propagation {
		case config.MountPropagationNone:
		case config.MountPropagationBidirectional:
			attrs = append(attrs, "rshared")
		case config.MountPropagationHostToContainer:
			attrs = append(attrs, "rslave")
		default:
		}
		if len(attrs) > 0 {
			bind = fmt.Sprintf("%s:%s", bind, strings.Join(attrs, ","))
		}
		args = append(args, fmt.Sprintf("--volume=%s", bind))
	}
	return args
}

func generatePortMappings(clusterIPFamily config.ClusterIPFamily, portMappings ...config.PortMapping) ([]string, error) {
	args := make([]string, 0, len(portMappings))
	for _, pm := range portMappings {
		if pm.ListenAddress == "" {
			switch clusterIPFamily {
			case config.IPv4Family, config.DualStackFamily:
				pm.ListenAddress = "0.0.0.0"
			case config.IPv6Family:
				pm.ListenAddress = "::"
			default:
				return nil, errors.Errorf("unknown cluster IP family: %v", clusterIPFamily)
			}
		}
		if string(pm.Protocol) == "" {
			pm.Protocol = config.PortMappingProtocolTCP
		}
		switch pm.Protocol {
		case config.PortMappingProtocolTCP, config.PortMappingProtocolUDP, config.PortMappingProtocolSCTP:
		default:
			return nil, errors.Errorf("unknown port mapping protocol: %v", pm.Protocol)
		}
		hostPort, releaseHostPortFn, err := common.PortOrGetFreePort(pm.HostPort, pm.ListenAddress)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get random host port for port mapping")
		}
		if releaseHostPortFn != nil {
			defer releaseHostPortFn()
		}
		protocol := string(pm.Protocol)
		hostPortBinding := net.JoinHostPort(pm.ListenAddress, fmt.Sprintf("%d", hostPort))
		args = append(args, fmt.Sprintf("--publish=%s:%d/%s", hostPortBinding, pm.ContainerPort, protocol))
	}
	return args, nil
}

func createContainerWithWaitUntilSystemdReachesMultiUserSystem(ctxName, name string, args []string) error {
	runArgs := dockerArgs(ctxName, "run", "--name", name)
	runArgs = append(runArgs, args...)
	if err := exec.Command("docker", runArgs...).Run(); err != nil {
		return err
	}

	logCtx, logCancel := context.WithTimeout(context.Background(), 30*time.Second)
	logCmd := exec.CommandContext(logCtx, "docker",
		dockerArgs(ctxName, "logs", "-f", name)...,
	)
	defer logCancel()
	return common.WaitUntilLogRegexpMatches(logCtx, logCmd, common.NodeReachedCgroupsReadyRegexp())
}
