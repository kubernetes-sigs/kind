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

// Package swarm is the kind provider implementation that distributes node
// containers across multiple Docker hosts on a shared Swarm overlay network.
package swarm

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/internal/sets"
)

// NewProvider returns a multi-host kind provider built on top of an existing
// (or about-to-be-bootstrapped) Docker Swarm.  hosts[0] is treated as the
// swarm manager; node containers are round-robined across hosts.
func NewProvider(logger log.Logger, hosts []Host, opts ...Option) providers.Provider {
	p := &provider{
		logger: logger,
		hosts:  hosts,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Option configures a swarm provider.
type Option func(*provider)

// WithBootstrap makes Provision run `docker swarm init` on the manager and
// `docker swarm join` on each worker before creating the cluster.
func WithBootstrap() Option { return func(p *provider) { p.bootstrap = true } }

// WithOverlay overrides the default overlay network name ("kind").
func WithOverlay(name string) Option { return func(p *provider) { p.overlay = name } }

type provider struct {
	logger    log.Logger
	hosts     []Host
	overlay   string
	bootstrap bool
	info      *providers.ProviderInfo
}

func (p *provider) String() string { return "swarm" }

func (p *provider) overlayName() string {
	if p.overlay != "" {
		return p.overlay
	}
	return swarmOverlayName
}

func (p *provider) manager() Host { return p.hosts[0] }

// Provision is part of the providers.Provider interface
func (p *provider) Provision(status *cli.Status, cfg *config.Cluster) (err error) {
	if len(p.hosts) == 0 {
		return errors.New("swarm provider: no hosts configured")
	}

	// optionally initialise the swarm
	if p.bootstrap {
		if err := initSwarmIfNeeded(p.manager(), p.hosts[1:]); err != nil {
			return errors.Wrap(err, "swarm bootstrap")
		}
	}

	// ensure the overlay exists on the manager
	if err := ensureSwarmOverlay(p.manager(), p.overlayName()); err != nil {
		return errors.Wrap(err, "failed to ensure swarm overlay")
	}

	// ensure node images on every host
	if err := ensureNodeImages(p.logger, status, cfg, p.hosts); err != nil {
		return err
	}

	icons := strings.Repeat("📦 ", len(cfg.Nodes))
	status.Start(fmt.Sprintf("Preparing nodes %s", icons))
	defer func() { status.End(err == nil) }()

	createContainerFuncs, err := planCreation(cfg, p.overlayName(), p.hosts)
	if err != nil {
		return err
	}
	return errors.UntilErrorConcurrent(createContainerFuncs)
}

// ListClusters is part of the providers.Provider interface.
// We query every configured host's daemon and union the cluster names.
func (p *provider) ListClusters() ([]string, error) {
	all := sets.NewString()
	for _, h := range p.hosts {
		cmd := exec.Command("docker",
			dockerArgs(h.Context,
				"ps", "-a",
				"--filter", "label="+clusterLabelKey,
				"--format", fmt.Sprintf(`{{.Label "%s"}}`, clusterLabelKey),
			)...,
		)
		lines, err := exec.OutputLines(cmd)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list clusters on %s", h.Context)
		}
		all.Insert(lines...)
	}
	return all.List(), nil
}

// ListNodes is part of the providers.Provider interface.
// We query every host and tag returned containers with their owning context.
func (p *provider) ListNodes(cluster string) ([]nodes.Node, error) {
	var ret []nodes.Node
	for _, h := range p.hosts {
		cmd := exec.Command("docker",
			dockerArgs(h.Context,
				"ps", "-a",
				"--filter", fmt.Sprintf("label=%s=%s", clusterLabelKey, cluster),
				"--format", `{{.Names}}`,
			)...,
		)
		lines, err := exec.OutputLines(cmd)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list nodes on %s", h.Context)
		}
		for _, name := range lines {
			ret = append(ret, p.node(name, h.Context))
		}
	}
	return ret, nil
}

// DeleteNodes is part of the providers.Provider interface.
// Containers are grouped by host (the node carries its host context) and
// removed with one `docker rm` per host.
func (p *provider) DeleteNodes(ns []nodes.Node) error {
	if len(ns) == 0 {
		return nil
	}
	byHost := map[string][]string{}
	for _, n := range ns {
		host := ""
		if sn, ok := n.(*node); ok {
			host = sn.host
		}
		if host == "" {
			host = p.manager().Context
		}
		byHost[host] = append(byHost[host], n.String())
	}
	for host, names := range byHost {
		args := dockerArgs(host, "rm", "-f", "-v")
		args = append(args, names...)
		if err := exec.Command("docker", args...).Run(); err != nil {
			return errors.Wrapf(err, "failed to delete nodes on %s", host)
		}
	}
	return nil
}

// GetAPIServerEndpoint is part of the providers.Provider interface.
func (p *provider) GetAPIServerEndpoint(cluster string) (string, error) {
	allNodes, err := p.ListNodes(cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to list nodes")
	}
	n, err := nodeutils.APIServerEndpointNode(allNodes)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server endpoint")
	}

	host := ""
	if sn, ok := n.(*node); ok {
		host = sn.host
	}
	if host == "" {
		host = p.manager().Context
	}

	cmd := exec.Command("docker",
		dockerArgs(host, "inspect",
			"--format", fmt.Sprintf(
				"{{ with (index (index .NetworkSettings.Ports \"%d/tcp\") 0) }}{{ printf \"%%s\t%%s\" .HostIp .HostPort }}{{ end }}",
				common.APIServerInternalPort,
			),
			n.String())...,
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server port")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("network details should only be one line, got %d lines", len(lines))
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) != 2 {
		return "", errors.Errorf("network details should only be two parts, got %d", len(parts))
	}
	// On swarm the published port is reachable on the host's external address;
	// prefer that over the 0.0.0.0 / :: that docker reports.
	hostAddr := parts[0]
	for _, h := range p.hosts {
		if h.Context == host && h.Addr != "" {
			hostAddr = h.Addr
			break
		}
	}
	return net.JoinHostPort(hostAddr, parts[1]), nil
}

// GetAPIServerInternalEndpoint is part of the providers.Provider interface.
func (p *provider) GetAPIServerInternalEndpoint(cluster string) (string, error) {
	allNodes, err := p.ListNodes(cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to list nodes")
	}
	n, err := nodeutils.APIServerEndpointNode(allNodes)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server endpoint")
	}
	return net.JoinHostPort(n.String(), fmt.Sprintf("%d", common.APIServerInternalPort)), nil
}

// node returns a new node handle for this provider, bound to a host.
func (p *provider) node(name, host string) nodes.Node {
	return &node{name: name, host: host}
}

// CollectLogs will populate dir with cluster logs and other debug files.
func (p *provider) CollectLogs(dir string, ns []nodes.Node) error {
	execToPathFn := func(cmd exec.Cmd, path string) func() error {
		return func() error {
			f, err := common.FileOnHost(path)
			if err != nil {
				return err
			}
			defer f.Close()
			return cmd.SetStdout(f).SetStderr(f).Run()
		}
	}
	fns := []func() error{}
	for _, h := range p.hosts {
		ctx := h.Context
		fns = append(fns, execToPathFn(
			exec.Command("docker", dockerArgs(ctx, "info")...),
			filepath.Join(dir, fmt.Sprintf("docker-info-%s.txt", ctx)),
		))
	}
	for _, n := range ns {
		host := p.manager().Context
		if sn, ok := n.(*node); ok && sn.host != "" {
			host = sn.host
		}
		name := n.String()
		path := filepath.Join(dir, name)
		fns = append(fns, execToPathFn(
			exec.Command("docker", dockerArgs(host, "inspect", name)...),
			filepath.Join(path, "inspect.json"),
		))
	}
	return errors.AggregateConcurrent(fns)
}

// Info returns the provider info, queried from the swarm manager.
func (p *provider) Info() (*providers.ProviderInfo, error) {
	var err error
	if p.info == nil {
		p.info, err = info(p.manager().Context)
	}
	return p.info, err
}

// dockerInfo corresponds to `docker info --format '{{json .}}'`
type dockerInfo struct {
	CgroupDriver    string   `json:"CgroupDriver"`
	CgroupVersion   string   `json:"CgroupVersion"`
	MemoryLimit     bool     `json:"MemoryLimit"`
	PidsLimit       bool     `json:"PidsLimit"`
	CPUShares       bool     `json:"CPUShares"`
	SecurityOptions []string `json:"SecurityOptions"`
}

func info(ctxName string) (*providers.ProviderInfo, error) {
	cmd := exec.Command("docker", dockerArgs(ctxName, "info", "--format", "{{json .}}")...)
	out, err := exec.Output(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get docker info on %s", ctxName)
	}
	var dInfo dockerInfo
	if err := json.Unmarshal(out, &dInfo); err != nil {
		return nil, err
	}
	pi := providers.ProviderInfo{
		Cgroup2: dInfo.CgroupVersion == "2",
	}
	if dInfo.CgroupDriver != "none" {
		pi.SupportsMemoryLimit = dInfo.MemoryLimit
		pi.SupportsPidsLimit = dInfo.PidsLimit
		pi.SupportsCPUShares = dInfo.CPUShares
	}
	for _, o := range dInfo.SecurityOptions {
		csvReader := csv.NewReader(strings.NewReader(o))
		sliceSlice, err := csvReader.ReadAll()
		if err != nil {
			return nil, err
		}
		for _, f := range sliceSlice {
			for _, ff := range f {
				if ff == "name=rootless" {
					pi.Rootless = true
				}
			}
		}
	}
	return &pi, nil
}
