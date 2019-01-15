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
	"reflect"
	"sort"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/config"
)

func TestExecutionPlanSorting(t *testing.T) {
	cases := []struct {
		TestName string
		actual   executionPlan
		expected executionPlan
	}{
		{
			TestName: "ExecutionPlan is ordered by provisioning order as a first criteria",
			actual: executionPlan{
				&plannedTask{Node: &NodeReplica{Name: "worker2", Node: config.Node{Role: config.WorkerRole}}},
				&plannedTask{Node: &NodeReplica{Name: "control-plane2", Node: config.Node{Role: config.ControlPlaneRole}}},
				&plannedTask{Node: &NodeReplica{Name: "etcd", Node: config.Node{Role: config.ExternalEtcdRole}}},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}},
			},
			expected: executionPlan{
				&plannedTask{Node: &NodeReplica{Name: "etcd", Node: config.Node{Role: config.ExternalEtcdRole}}},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}},
				&plannedTask{Node: &NodeReplica{Name: "control-plane2", Node: config.Node{Role: config.ControlPlaneRole}}},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}},
				&plannedTask{Node: &NodeReplica{Name: "worker2", Node: config.Node{Role: config.WorkerRole}}},
			},
		},
		{
			TestName: "ExecutionPlan respects the given action order as a second criteria",
			actual: executionPlan{
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 3},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 2},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 1},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 1},
			},
			expected: executionPlan{
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 1},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 2},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 1},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 3},
			},
		},
		{
			TestName: "ExecutionPlan respects the predefined order for each action as a third criteria",
			actual: executionPlan{
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 1, taskIndex: 2},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 1, taskIndex: 2},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 1, taskIndex: 1},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 1, taskIndex: 1},
			},
			expected: executionPlan{
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 1, taskIndex: 1},
				&plannedTask{Node: &NodeReplica{Name: "control-plane1", Node: config.Node{Role: config.ControlPlaneRole}}, actionIndex: 1, taskIndex: 2},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 1, taskIndex: 1},
				&plannedTask{Node: &NodeReplica{Name: "worker1", Node: config.Node{Role: config.WorkerRole}}, actionIndex: 1, taskIndex: 2},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t2 *testing.T) {
			// sorting planned task
			sort.Sort(c.actual)

			// cheching planned tasks are properly sorted
			if !reflect.DeepEqual(c.actual, c.expected) {
				t2.Errorf("Expected machineSets")
				for _, m := range c.expected {
					t2.Logf("	%s on %s, actionIndex %d taskIndex %d", m.Task.Description, m.Node.Name, m.actionIndex, m.taskIndex)
				}
				t2.Log("Saw")
				for _, m := range c.actual {
					t2.Logf("	%s on %s, actionIndex %d taskIndex %d", m.Task.Description, m.Node.Name, m.actionIndex, m.taskIndex)
				}
			}
		})
	}
}

// dummy action with single task targeting all nodes
type action0 struct{}

func newAction0() Action {
	return &action0{}
}

func (b *action0) Tasks() []Task {
	return []Task{
		{
			Description: "action0 - task 0/all",
			TargetNodes: selectAllNodes,
		},
	}
}

// dummy action with single task targeting control-plane nodes
type action1 struct{}

func newAction1() Action {
	return &action1{}
}

func (b *action1) Tasks() []Task {
	return []Task{
		{
			Description: "action1 - task 0/control-planes",
			TargetNodes: selectControlPlaneNodes,
		},
	}
}

// dummy action with multiple tasks each with different targets
type action2 struct{}

func newAction2() Action {
	return &action2{}
}

func (b *action2) Tasks() []Task {
	return []Task{
		{
			Description: "action2 - task 0/all",
			TargetNodes: selectAllNodes,
		},
		{
			Description: "action2 - task 1/control-planes",
			TargetNodes: selectControlPlaneNodes,
		},
		{
			Description: "action2 - task 2/workers",
			TargetNodes: selectWorkerNodes,
		},
	}
}

func TestNewExecutionPlan(t *testing.T) {
	testTopology := []*config.Node{
		{Role: config.ControlPlaneRole}, // 1 control-plane
		{Role: config.WorkerRole},       // 2 workers
		{Role: config.WorkerRole},
	}

	registerAction("action0", newAction0) // Task 0 -> allMachines
	registerAction("action1", newAction1) // Task 0 -> controlPlaneMachines
	registerAction("action2", newAction2) // Task 0 -> allMachines, Task 1 -> controlPlaneMachines, Task 2 -> workerMachines

	cases := []struct {
		TestName     string
		Actions      []string
		Nodes        []*config.Node
		ExpextedPlan []string
	}{
		{
			TestName: "Action with task targeting all machines is planned",
			Actions:  []string{"action0"},
			Nodes:    testTopology,
			ExpextedPlan: []string{
				"action0 - task 0/all on control-plane",
				"action0 - task 0/all on worker1",
				"action0 - task 0/all on worker2",
			},
		},
		{
			TestName: "Action with task targeting control-plane nodes is planned",
			Actions:  []string{"action1"},
			Nodes:    testTopology,
			ExpextedPlan: []string{
				"action1 - task 0/control-planes on control-plane",
			},
		},
		{
			TestName: "Action with many task and targets is planned",
			Actions:  []string{"action2"},
			Nodes:    testTopology,
			ExpextedPlan: []string{ // task are grouped by machine/provision order and task order is preserved
				"action2 - task 0/all on control-plane",
				"action2 - task 1/control-planes on control-plane",
				"action2 - task 0/all on worker1",
				"action2 - task 2/workers on worker1",
				"action2 - task 0/all on worker2",
				"action2 - task 2/workers on worker2",
			},
		},
		{
			TestName: "Many actions are planned",
			Actions:  []string{"action0", "action1", "action2"},
			Nodes:    testTopology,
			ExpextedPlan: []string{ // task are grouped by machine/provision order and action order/task order is preserved
				"action0 - task 0/all on control-plane",
				"action1 - task 0/control-planes on control-plane",
				"action2 - task 0/all on control-plane",
				"action2 - task 1/control-planes on control-plane",
				"action0 - task 0/all on worker1",
				"action2 - task 0/all on worker1",
				"action2 - task 2/workers on worker1",
				"action0 - task 0/all on worker2",
				"action2 - task 0/all on worker2",
				"action2 - task 2/workers on worker2",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t2 *testing.T) {
			var derived = &DerivedConfig{}
			// Adding nodes to the config
			for _, n := range c.Nodes {
				if err := derived.Add(n); err != nil {
					t2.Fatalf("unexpected error while adding nodes: %v", err)
					break
				}
			}
			// Creating the execution plane
			tasks, _ := newExecutionPlan(derived, c.Actions)

			// Checking planned task are properly created (and sorted)
			if len(tasks) != len(c.ExpextedPlan) {
				t2.Fatalf("Invalid PlannedTask expected %d elements, saw %d", len(c.ExpextedPlan), len(tasks))
			}

			for i, mt := range tasks {
				r := fmt.Sprintf("%s on %s", mt.Task.Description, mt.Node.Name)
				if r != c.ExpextedPlan[i] {
					t2.Errorf("Invalid PlannedTask %d expected %v, saw %v", i, c.ExpextedPlan[i], r)
				}
			}
		})
	}
}
