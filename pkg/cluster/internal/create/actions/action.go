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

package actions

import (
	"fmt"
	"sync"

	"sigs.k8s.io/kind/pkg/cluster/internal/context"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/log"

	simpleActions "gitlab.com/digitalxero/simple-actions"
	"gitlab.com/digitalxero/simple-actions/term"
)

type actionContext struct {
	logger log.Logger
	status *term.Status
	dryRun bool
	data   *ActionContextData
}

func (ac *actionContext) Logger() log.Logger {
	return ac.logger
}

func (ac *actionContext) Status() *term.Status {
	return ac.status
}

func (ac *actionContext) IsDryRun() bool {
	return ac.dryRun
}

func (ac *actionContext) Data() interface{} {
	return ac.data
}

func Data(ctx simpleActions.ActionContext) (data *ActionContextData, err error) {
	var ok bool
	data, ok = ctx.Data().(*ActionContextData)
	if !ok {
		return nil, fmt.Errorf("unable to convert ctx data to required struct")
	}

	return data, nil
}

// ActionContext is data supplied to all actions
type ActionContextData struct {
	Config         *config.Cluster
	ClusterContext *context.Context
	cache          *cachedData
}

// NewActionContext returns a new ActionContext
func NewActionContext(
	logger log.Logger,
	cfg *config.Cluster,
	ctx *context.Context,
	dryRun bool,
) simpleActions.ActionContext {
	return &actionContext{
		logger: logger,
		status: term.StatusForLogger(logger),
		dryRun: dryRun,
		data: &ActionContextData{
			Config:         cfg,
			ClusterContext: ctx,
			cache:          &cachedData{},
		},
	}
}

type cachedData struct {
	mu    sync.RWMutex
	nodes []nodes.Node
}

func (cd *cachedData) getNodes() []nodes.Node {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return cd.nodes
}

func (cd *cachedData) setNodes(n []nodes.Node) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	cd.nodes = n
}

// Nodes returns the list of cluster nodes, this is a cached call
func (ac *ActionContextData) Nodes() ([]nodes.Node, error) {
	cachedNodes := ac.cache.getNodes()
	if cachedNodes != nil {
		return cachedNodes, nil
	}
	n, err := ac.ClusterContext.ListNodes()
	if err != nil {
		return nil, err
	}
	ac.cache.setNodes(n)
	return n, nil
}
