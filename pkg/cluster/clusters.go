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
	"sigs.k8s.io/kind/pkg/internal/cluster/providers/docker"
)

// List returns a list of clusters for which node containers exist
// TODO: refactor this to not assume a particular provider
func List() ([]string, error) {
	return docker.NewProvider().ListClusters()
}

// IsKnown return true if a cluster exists with the given name.
// If obtaining the list of known clusters fails the function returns an error.
func IsKnown(name string) (bool, error) {
	list, err := List()
	if err != nil {
		return false, err
	}
	for _, cluster := range list {
		if cluster == name {
			return true, nil
		}
	}
	return false, nil
}
