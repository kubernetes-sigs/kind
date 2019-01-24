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

package create

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Action define a set of tasks to be executed on a `kind` cluster.
// TODO(bentheelder): redesign this for concurrency
// Usage of actions allows to define repetitive, high level abstractions/workflows
// by composing lower level tasks
type Action interface {
	// Tasks returns the list of task that are identified by this action
	// Please note that the order of task is important, and it will be
	// respected during execution
	Tasks() []Task
}

// Task define a logical step of an action to be executed on a `kind` cluster.
// At exec time the logical step will then apply to the current cluster
// topology, and be planned for execution zero, one or many times accordingly.
type Task struct {
	// Description of the task
	Description string
	// TargetNodes define a function that identifies the nodes where this
	// task should be executed
	TargetNodes nodeSelector
	// Run the func that implements the task action
	Run func(*execContext, *NodeReplica) error
}

// execContext is a superset of Context used by helpers for Context.Exec() command
// TODO(bentheelder): refactor this
type execContext struct {
	*Context
	status *logutil.Status
	// nodes contains the list of actual nodes (a node is a container implementing a config node)
	nodes        map[string]*nodes.Node
	waitForReady time.Duration // Wait for the control plane node to be ready
}

func (ec *execContext) NodeFor(configNode *NodeReplica) (node *nodes.Node, ok bool) {
	node, ok = ec.nodes[configNode.Name]
	return
}

// nodeSelector defines a function returning a subset of nodes where tasks
// should be planned.
type nodeSelector func(*DerivedConfig) ReplicaList

// plannedTask defines a Task planned for execution on a given node.
type plannedTask struct {
	// task to be executed
	Task Task
	// node where the task should be executed
	Node *NodeReplica

	// PlannedTask should respects the given order of actions and tasks
	actionIndex int
	taskIndex   int
}

// executionPlan contain an ordered list of Planned Tasks
// Please note that the planning order is critical for providing a
// predictable, "kubeadm friendly" and consistent execution order.
type executionPlan []*plannedTask

// internal registry of named Action implementations
var actionImpls = struct {
	impls map[string]func() Action
	sync.Mutex
}{
	impls: map[string]func() Action{},
}

// registerAction registers a new named actionBuilder function for use
func registerAction(name string, actionBuilderFunc func() Action) {
	actionImpls.Lock()
	actionImpls.impls[name] = actionBuilderFunc
	actionImpls.Unlock()
}

// getAction returns one instance of a registered action
func getAction(name string) (Action, error) {
	actionImpls.Lock()
	actionBuilderFunc, ok := actionImpls.impls[name]
	actionImpls.Unlock()
	if !ok {
		return nil, errors.Errorf("no Action implementation with name: %s", name)
	}
	return actionBuilderFunc(), nil
}

// newExecutionPlan creates an execution plan by applying logical step/task
// defined for each action to the actual cluster topology. As a result task
// could be executed zero, one or more times according with the target nodes
// selector defined for each task.
// The execution plan is ordered, providing a predictable, "kubeadm friendly"
// and consistent execution order; with this regard please note that the order
// of actions is important, and it will be respected by planning.
// TODO(fabrizio pandini): probably it will be necessary to add another criteria
//     for ordering planned task for the most complex workflows (e.g.
//     init-join-upgrade and then join again)
//     e.g. it should be something like "action group" where each action
//	   group is a list of actions
func newExecutionPlan(cfg *DerivedConfig, actionNames []string) (executionPlan, error) {
	// for each actionName
	var plan = executionPlan{}
	for i, name := range actionNames {
		// get the action implementation instance
		actionImpl, err := getAction(name)
		if err != nil {
			return nil, err
		}
		// for each logical tasks defined for the action
		for j, t := range actionImpl.Tasks() {
			// get the list of target nodes in the current topology
			targetNodes := t.TargetNodes(cfg)
			for _, n := range targetNodes {
				// creates the planned task
				taskContext := &plannedTask{
					Node:        n,
					Task:        t,
					actionIndex: i,
					taskIndex:   j,
				}
				plan = append(plan, taskContext)
			}
		}
	}

	// sorts the list of planned task ensuring a predictable, "kubeadm friendly"
	// and consistent execution order
	sort.Sort(plan)
	return plan, nil
}

// Len of the executionPlan.
// It is required for making ExecutionPlan sortable.
func (t executionPlan) Len() int {
	return len(t)
}

// Less return the lower between two elements of the ExecutionPlan, where the
// lower element should be executed before the other.
// It is required for making ExecutionPlan sortable.
func (t executionPlan) Less(i, j int) bool {
	return t[i].ExecutionOrder() < t[j].ExecutionOrder()
}

// ExecutionOrder returns a string that can be used for sorting planned tasks
// into a predictable, "kubeadm friendly" and consistent order.
// NB. we are using a string to combine all the item considered into something
// that can be easily sorted using a lexicographical order
func (p *plannedTask) ExecutionOrder() string {
	return fmt.Sprintf("Node.ProvisioningOrder: %d - Node.Name: %s - actionIndex: %d - taskIndex: %d",
		// Then PlannedTask are grouped by machines, respecting the kubeadm node
		// ProvisioningOrder: first complete provisioning on bootstrap control
		// plane, then complete provisioning of secondary control planes, and
		// finally provision worker nodes.
		p.Node.ProvisioningOrder(),
		// Node name is considered in order to get a predictable/repeatable ordering
		// in case of many nodes with the same ProvisioningOrder
		p.Node.Name,
		// If both the two criteria above are equal, the given order of actions will
		// be respected and, for each action, the predefined order of tasks
		// will be used
		p.actionIndex,
		p.taskIndex,
	)
}

// Swap two elements of the ExecutionPlan.
// It is required for making ExecutionPlan sortable.
func (t executionPlan) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// selectAllNodes is a NodeSelector that returns all the nodes defined in
// the `kind` Config
func selectAllNodes(cfg *DerivedConfig) ReplicaList {
	return cfg.AllReplicas()
}

// selectControlPlaneNodes is a NodeSelector that returns all the nodes
// with control-plane role
func selectControlPlaneNodes(cfg *DerivedConfig) ReplicaList {
	return cfg.ControlPlanes()
}

// selectBootstrapControlPlaneNode is a NodeSelector that returns the
// first node with control-plane role
func selectBootstrapControlPlaneNode(cfg *DerivedConfig) ReplicaList {
	if cfg.BootStrapControlPlane() != nil {
		return ReplicaList{cfg.BootStrapControlPlane()}
	}
	return nil
}

// selectSecondaryControlPlaneNodes is a NodeSelector that returns all
// the nodes with control-plane roleexcept the BootStrapControlPlane
// node, if any,
func selectSecondaryControlPlaneNodes(cfg *DerivedConfig) ReplicaList {
	return cfg.SecondaryControlPlanes()
}

// selectWorkerNodes is a NodeSelector that returns all the nodes with
// Worker role, if any
func selectWorkerNodes(cfg *DerivedConfig) ReplicaList {
	return cfg.Workers()
}

// selectExternalEtcdNode is a NodeSelector that returns the node with
//external-etcd role, if defined
func selectExternalEtcdNode(cfg *DerivedConfig) ReplicaList {
	if cfg.ExternalEtcd() != nil {
		return ReplicaList{cfg.ExternalEtcd()}
	}
	return nil
}

// selectExternalLoadBalancerNode is a NodeSelector that returns the node
// with external-load-balancer role, if defined
func selectExternalLoadBalancerNode(cfg *DerivedConfig) ReplicaList {
	if cfg.ExternalLoadBalancer() != nil {
		return ReplicaList{cfg.ExternalLoadBalancer()}
	}
	return nil
}
