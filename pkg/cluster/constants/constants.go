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

// Package constants contains well known constants for kind clusters
package constants

// ClusterLabelKey is applied to each "node" docker container for identification
const ClusterLabelKey = "io.k8s.sigs.kind.cluster"

// ClusterRoleKey is applied to each "node" docker container for categorization of nodes by role
const ClusterRoleKey = "io.k8s.sigs.kind.role"
