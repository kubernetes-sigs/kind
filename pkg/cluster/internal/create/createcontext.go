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
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/internal/context"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Context is a superset of the main cluster Context implementing helpers internal
// to the user facing Context.Create()
// TODO(bentheelder): elminate this object
type Context struct {
	*context.Context
	// other fields
	Status *logutil.Status
	Config *config.Config
	*DerivedConfig
	Retain      bool         // if we should retain nodes after failing to create.
	ExecOptions []ExecOption // options to be forwarded to the exec command.
}

//ExecOption is an execContext configuration option supplied to Exec
type ExecOption func(*execContext)

// WaitForReady configures execContext to use interval as maximum wait time for the control plane node to be ready
func WaitForReady(interval time.Duration) ExecOption {
	return func(c *execContext) {
		c.waitForReady = interval
	}
}

// Exec actions on kubernetes-in-docker cluster
// TODO(bentheelder): refactor this further
// Actions are repetitive, high level abstractions/workflows composed
// by one or more lower level tasks, that automatically adapt to the
// current cluster topology
func (cc *Context) Exec(nodeList map[string]*nodes.Node, actions []string, options ...ExecOption) error {
	// init the exec context and logging
	ec := &execContext{
		Context: cc,
		nodes:   nodeList,
	}

	ec.status = logutil.NewStatus(os.Stdout)
	ec.status.MaybeWrapLogrus(log.StandardLogger())

	defer ec.status.End(false)

	// apply exec options
	for _, option := range options {
		option(ec)
	}

	// Create an ExecutionPlan that applies the given actions to the
	// topology defined in the config
	executionPlan, err := newExecutionPlan(ec.DerivedConfig, actions)
	if err != nil {
		return err
	}

	// Executes all the selected action
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
