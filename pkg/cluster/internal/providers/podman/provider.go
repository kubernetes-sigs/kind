/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliep.
See the License for the specific language governing permissions and
limitations under the License.
*/

package podman

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	internallogs "sigs.k8s.io/kind/pkg/cluster/internal/logs"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/provider"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/provider/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cli"
)

// NewProvider returns a new provider based on executing `podman ...`
func NewProvider(logger log.Logger) provider.Provider {
	logger.Warn("enabling experimental podman provider")
	return &Provider{
		logger: logger,
	}
}

// Provider implements provider.Provider
// see NewProvider
type Provider struct {
	logger log.Logger
}

// Provision is part of the providers.Provider interface
func (p *Provider) Provision(status *cli.Status, cluster string, cfg *config.Cluster) (err error) {
	if err := ensureMinVersion(); err != nil {
		return err
	}

	// kind doesn't currently work with podman rootless, surface a warning
	if os.Geteuid() != 0 {
		p.logger.Warn("podman provider may not work properly in rootless mode")
	}

	// TODO: validate cfg
	// ensure node images are pulled before actually provisioning
	if err := ensureNodeImages(p.logger, status, cfg); err != nil {
		return err
	}

	// actually provision the cluster
	icons := strings.Repeat("📦 ", len(cfg.Nodes))
	status.Start(fmt.Sprintf("Preparing nodes %s", icons))
	defer func() { status.End(err == nil) }()

	// plan creating the containers
	createContainerFuncs, err := planCreation(cluster, cfg)
	if err != nil {
		return err
	}

	// actually create nodes
	return errors.UntilErrorConcurrent(createContainerFuncs)
}

// ListClusters is part of the providers.Provider interface
func (p *Provider) ListClusters() ([]string, error) {
	cmd := exec.Command("podman",
		"ps",
		"-a", // show stopped nodes
		// filter for nodes with the cluster label
		"--filter", "label="+clusterLabelKey,
		// format to include the cluster name
		"--format", fmt.Sprintf(`{{index .Labels "%s"}}`, clusterLabelKey),
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clusters")
	}
	return sets.NewString(lines...).List(), nil
}

// ListNodes is part of the providers.Provider interface
func (p *Provider) ListNodes(cluster string) ([]nodes.Node, error) {
	cmd := exec.Command("podman",
		"ps",
		"-a", // show stopped nodes
		// filter for nodes with the cluster label
		"--filter", fmt.Sprintf("label=%s=%s", clusterLabelKey, cluster),
		// format to include the cluster name
		"--format", `{{.Names}}`,
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clusters")
	}
	// convert names to node handles
	ret := make([]nodes.Node, 0, len(lines))
	for _, name := range lines {
		ret = append(ret, p.node(name))
	}
	return ret, nil
}

// DeleteNodes is part of the providers.Provider interface
func (p *Provider) DeleteNodes(n []nodes.Node) error {
	if len(n) == 0 {
		return nil
	}
	const command = "podman"
	args := make([]string, 0, len(n)+3) // allocate once
	args = append(args,
		"rm",
		"-f", // force the container to be delete now
		"-v", // delete volumes
	)
	for _, node := range n {
		args = append(args, node.String())
	}
	if err := exec.Command(command, args...).Run(); err != nil {
		return errors.Wrap(err, "failed to delete nodes")
	}
	return nil
}

// GetAPIServerEndpoint is part of the providers.Provider interface
func (p *Provider) GetAPIServerEndpoint(cluster string) (string, error) {
	// locate the node that hosts this
	allNodes, err := p.ListNodes(cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to list nodes")
	}
	n, err := nodeutils.APIServerEndpointNode(allNodes)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server endpoint")
	}

	// retrieve the specific port mapping using podman inspect
	cmd := exec.Command(
		"podman", "inspect",
		"--format",
		"{{ json .NetworkSettings.Ports }}",
		n.String(),
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server port")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("network details should only be one line, got %d lines", len(lines))
	}

	// portMapping maps to the standard CNI portmapping capability
	// see: https://github.com/containernetworking/cni/blob/spec-v0.4.0/CONVENTIONS.md
	type portMapping struct {
		HostPort      int32  `json:"hostPort"`
		ContainerPort int32  `json:"containerPort"`
		Protocol      string `json:"protocol"`
		HostIP        string `json:"hostIP"`
	}

	var portMappings []portMapping
	if err := json.Unmarshal([]byte(lines[0]), &portMappings); err != nil {
		return "", errors.Errorf("invalid network details: %v", err)
	}
	for _, pm := range portMappings {
		if pm.ContainerPort == common.APIServerInternalPort && pm.Protocol == "tcp" {
			return net.JoinHostPort(pm.HostIP, strconv.Itoa(int(pm.HostPort))), nil
		}
	}

	return "", errors.Errorf("unable to find apiserver endpoint information")
}

// node returns a new node handle for this provider
func (p *Provider) node(name string) nodes.Node {
	return &node{
		name: name,
	}
}

// CollectLogs will populate dir with cluster logs and other debug files
func (p *Provider) CollectLogs(dir string, nodes []nodes.Node) error {
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
	// construct a slice of methods to collect logs
	fns := []func() error{
		// TODO(bentheelder): record the kind version here as well
		// record info about the host podman
		execToPathFn(
			exec.Command("podman", "info"),
			filepath.Join(dir, "podman-info.txt"),
		),
	}

	// collect /var/log for each node and plan collecting more logs
	var errs []error
	for _, n := range nodes {
		node := n // https://golang.org/doc/faq#closures_and_goroutines
		name := node.String()
		path := filepath.Join(dir, name)
		if err := internallogs.DumpDir(p.logger, node, "/var/log", path); err != nil {
			errs = append(errs, err)
		}

		fns = append(fns,
			func() error { return common.CollectLogs(node, path) },
			execToPathFn(exec.Command("podman", "inspect", name), filepath.Join(path, "inspect.json")),
			func() error {
				f, err := common.FileOnHost(filepath.Join(path, "serial.log"))
				if err != nil {
					return err
				}
				return node.SerialLogs(f)
			},
		)
	}

	// run and collect up all errors
	errs = append(errs, errors.AggregateConcurrent(fns))
	return errors.NewAggregate(errs)
}
