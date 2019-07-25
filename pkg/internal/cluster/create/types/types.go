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

// Package types contains types is used to export and share create cluster
// options with multiple packages without creating cycles
package types

import (
	"time"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// ClusterOptions holds cluster creation options
// NOTE: this is only exported for usage by the parent package and the options
// package
// See ClusterOption instead
type ClusterOptions struct {
	Config *config.Cluster
	// NodeImage overrides the nodes' images in Config if non-zero
	NodeImage    string
	Retain       bool
	WaitForReady time.Duration
	//TODO: Refactor this. It is a temporary solution for a phased breakdown of different
	//      operations, specifically create. see https://github.com/kubernetes-sigs/kind/issues/324
	SetupKubernetes bool // if kind should setup kubernetes after creating nodes
}
