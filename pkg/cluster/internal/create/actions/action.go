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
	"sync"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/internal/context"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// Action defines a step of bringing up a kind cluster after initial node
// container creation
type Action interface {
	Execute(ctx *ActionContext) error
}

// ActionContext is data supplied to all actions
type ActionContext struct {
	Status         *logutil.Status
	Config         *config.Cluster
	ClusterContext *context.Context
	cache          *cachedData
}

// NewActionContext returns a new ActionContext
func NewActionContext(
	cfg *config.Cluster,
	ctx *context.Context,
	status *logutil.Status,
) *ActionContext {
	return &ActionContext{
		Status:         status,
		Config:         cfg,
		ClusterContext: ctx,
		cache:          &cachedData{},
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
func (ac *ActionContext) Nodes() ([]nodes.Node, error) {
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
