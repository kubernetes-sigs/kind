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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/consts"
	"sigs.k8s.io/kind/pkg/cluster/logs"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Context is used to create / manipulate kubernetes-in-docker clusters
// See: NewContext()
type Context struct {
	name string
}

// similar to valid docker container names, but since we will prefix
// and suffix this name, we can relax it a little
// see NewContext() for usage
// https://godoc.org/github.com/docker/docker/daemon/names#pkg-constants
var validNameRE = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// DefaultName is the default Context name
// TODO(bentheelder): consider removing automatic prefixing in favor
// of letting the user specify the full name..
const DefaultName = "1"

// NewContext returns a new cluster management context
// if name is "" the default ("1") will be used
func NewContext(name string) *Context {
	if name == "" {
		name = DefaultName
	}
	return &Context{
		name: name,
	}
}

// Validate will be called before creating new resources using the context
// It will not be called before deleting or listing resources, so as to allow
// contexts based around previously valid values to be used in newer versions
// You can call this early yourself to check validation before creation calls,
// though it will be called internally.
func (c *Context) Validate() error {
	// validate the name
	if !validNameRE.MatchString(c.name) {
		return fmt.Errorf(
			"'%s' is not a valid cluster name, cluster names must match `%s`",
			c.name, validNameRE.String(),
		)
	}
	return nil
}

// ControlPlaneMeta tracks various outputs that are relevant to the control plane created with Kind.
// Here we can define things like ports and listen or bind addresses as needed.
type ControlPlaneMeta struct {
	// APIServerPort is the port that the container is forwarding to the
	// Kubernetes API server running in the container
	APIServerPort int
}

// GetControlPlaneMeta attempts to retreive / compute metadata about
// the control plane for the context's cluster
// NOTE: due to refactoring this is currently non-functional (!)
// TODO(bentheelder): fix this
func (c *Context) GetControlPlaneMeta() (*ControlPlaneMeta, error) {
	return nil, fmt.Errorf("needs-reimplementation")
}

// ClusterLabel returns the docker object label that will be applied
// to cluster "node" containers
func (c *Context) ClusterLabel() string {
	return fmt.Sprintf("%s=%s", consts.ClusterLabelKey, c.name)
}

// Name returns the context's name
func (c *Context) Name() string {
	return c.name
}

// ClusterName returns the Kubernetes cluster name based on the context name
// currently this is .Name prefixed with "kind-"
func (c *Context) ClusterName() string {
	return fmt.Sprintf("kind-%s", c.name)
}

// KubeConfigPath returns the path to where the Kubeconfig would be placed
// by kind based on the configuration.
func (c *Context) KubeConfigPath() string {
	// TODO(bentheelder): Windows?
	// configDir matches the standard directory expected by kubectl etc
	configDir := filepath.Join(os.Getenv("HOME"), ".kube")
	// note that the file name however does not, we do not want to overwrite
	// the standard config, though in the future we may (?) merge them
	fileName := fmt.Sprintf("kind-config-%s", c.name)
	return filepath.Join(configDir, fileName)
}

// execContext is a superset of Context used by helpers for Context.Create()
// and Context.Exec() command
// TODO(fabrizio pandini): might be we want to move all the actions in a separated
//		package e.g. pkg/cluster/actions
//		In order to do this a circular dependency should be avoided:
//			pkg/cluster -- use -- pkg/cluster/actions
// 			pkg/cluster/actions -- use pkg/cluster execContext
type execContext struct {
	*Context
	status  *logutil.Status
	config  *config.Config
	derived *derivedConfigData
	// nodes contains the list of actual nodes (a node is a container implementing a config node)
	nodes        map[string]*nodes.Node
	waitForReady time.Duration // Wait for the control plane node to be ready
}

// Create provisions and starts a kubernetes-in-docker cluster
func (c *Context) Create(cfg *config.Config, retain bool, wait time.Duration) error {
	// validate config first
	if err := cfg.Validate(); err != nil {
		return err
	}

	// derive info necessary for creation
	derived, err := deriveInfo(cfg)
	if err != nil {
		return err
	}
	// validate node configuration
	if err := derived.Validate(); err != nil {
		return err
	}
	// TODO(fabrizio pandini): this check is temporary / WIP
	// kind v1alpha config fully supports multi nodes, but the cluster creation logic implemented in
	// pkg/cluster/contex.go does it only partially (yet).
	// As soon a external load-balancer and external etcd is implemented in pkg/cluster, this should go away
	if derived.ExternalLoadBalancer() != nil {
		return fmt.Errorf("multi node support is still a work in progress, currently external load balancer node is not supported")
	}
	if derived.SecondaryControlPlanes() != nil {
		return fmt.Errorf("multi node support is still a work in progress, currently only single control-plane node are supported")
	}
	if derived.ExternalEtcd() != nil {
		return fmt.Errorf("multi node support is still a work in progress, currently external etcd node is not supported")
	}

	fmt.Printf("Creating cluster '%s' ...\n", c.ClusterName())

	// init the create context and logging
	cc := &createContext{
		Context: c,
		config:  cfg,
		derived: derived,
		retain:  retain,
	}

	cc.status = logutil.NewStatus(os.Stdout)
	cc.status.MaybeWrapLogrus(log.StandardLogger())

	defer cc.status.End(false)

	// attempt to explicitly pull the required node images if they doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	cc.EnsureNodeImages()

	// Create node containers implementing defined config Nodes
	nodeList, err := cc.provisionNodes()
	if err != nil {
		// In case of errors nodes are deleted (except if retain is explicitly set)
		log.Error(err)
		if !cc.retain {
			cc.Delete()
		}
		return err
	}
	cc.status.End(true)

	// After creating node containers the Kubernetes provisioning is executed
	// By default `kind` executes all the actions required to get a fully working
	// Kubernetes cluster; please note that the list of actions automatically
	// adapt to the topology defined in config
	// TODO(fabrizio pandini): make the list of executed actions configurable from CLI
	err = c.exec(cc.config, cc.derived, nodeList, []string{"config", "init", "join"}, wait)
	if err != nil {
		// In case of errors nodes are deleted (except if retain is explicitly set)
		log.Error(err)
		if !cc.retain {
			cc.Delete()
		}
		return err
	}

	fmt.Printf(
		"Cluster creation complete. You can now use the cluster with:\n\nexport KUBECONFIG=\"$(kind get kubeconfig-path --name=%q)\"\nkubectl cluster-info\n",
		cc.Name(),
	)
	return nil
}

// TODO(bentheelder): refactor this
// Exec actions on kubernetes-in-docker cluster
// Actions are repetitive, high level abstractions/workflows composed
// by one or more lower level tasks, that automatically adapt to the
// current cluster topology
func (c *Context) exec(cfg *config.Config, derived *derivedConfigData, nodeList map[string]*nodes.Node, actions []string, wait time.Duration) error {
	// validate config first
	if err := cfg.Validate(); err != nil {
		return err
	}

	// init the exec context and logging
	ec := &execContext{
		Context:      c,
		config:       cfg,
		derived:      derived,
		nodes:        nodeList,
		waitForReady: wait,
	}

	ec.status = logutil.NewStatus(os.Stdout)
	ec.status.MaybeWrapLogrus(log.StandardLogger())

	defer ec.status.End(false)

	// Create an ExecutionPlan that applies the given actions to the topology defined
	// in the config
	executionPlan, err := newExecutionPlan(ec.derived, actions)
	if err != nil {
		return err
	}

	// Executes all the selected action
	// TODO(fabrizio pandini): add a flag to a filter PlannedTask by node
	// (e.g. execute only on this node) or by other criteria tbd
	for _, plannedTask := range executionPlan {
		ec.status.Start(fmt.Sprintf("[%s] %s", plannedTask.Node.Name, plannedTask.Task.Description))

		err := plannedTask.Task.Run(ec, plannedTask.Node)
		if err != nil {
			// in case of error, the execution plan is halted
			log.Error(err)
			return err
		}
	}
	ec.status.End(true)

	return nil
}

func (ec *execContext) NodeFor(configNode *nodeReplica) (node *nodes.Node, ok bool) {
	node, ok = ec.nodes[configNode.Name]
	return
}

// Delete tears down a kubernetes-in-docker cluster
func (c *Context) Delete() error {
	n, err := c.ListNodes()
	if err != nil {
		return fmt.Errorf("error listing nodes: %v", err)
	}

	// try to remove the kind kube config file generated by "kind create cluster"
	err = os.Remove(c.KubeConfigPath())
	if err != nil && !os.IsNotExist(err) {
		log.Warningf("Tried to remove %s but received error: %s\n", c.KubeConfigPath(), err)
	}

	// check if $KUBECONFIG is set and let the user know to unset if so
	if strings.Contains(os.Getenv("KUBECONFIG"), c.KubeConfigPath()) {
		fmt.Printf("$KUBECONFIG is still set to use %s even though that file has been deleted, remember to unset it\n", c.KubeConfigPath())
	}

	return nodes.Delete(n...)
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
func (c *Context) ListNodes() ([]nodes.Node, error) {
	return nodes.List("label=" + c.ClusterLabel())
}

// CollectLogs will populate dir with cluster logs and other debug files
func (c *Context) CollectLogs(dir string) error {
	nodes, err := c.ListNodes()
	if err != nil {
		return err
	}
	return logs.Collect(nodes, dir)
}
