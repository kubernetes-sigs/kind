/*
Copyright The Kubernetes Authors.

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

package images

import (
	"sigs.k8s.io/kind/pkg/cluster/images/layeraware"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/log"
)

// LoadResult contains outcome of loading
type LoadResult struct {
	Node    nodes.Node
	Success bool
	Error   error
}

// LoadImageLayerAware loads images with layer-aware transfer
//
// Supports multiple images in the tar (like current kind)
func LoadImageLayerAware(
	archivePath string,
	nodes []nodes.Node,
	logger log.Logger,
) ([]*LoadResult, error) {
	// Inspect once
	metadata, err := layeraware.InspectArchive(archivePath)
	if err != nil {
		return nil, err
	}

	// Load to each node
	results := make([]*LoadResult, len(nodes))
	for i, node := range nodes {
		result := &LoadResult{
			Node: node,
		}

		// Plan transfer
		plan, err := layeraware.PlanTransfer(metadata, node)
		if err != nil {
			result.Error = err
			results[i] = result
			continue
		}

		// Execute transfer
		err = layeraware.ExecuteTransfer(archivePath, plan, logger)
		if err != nil {
			result.Error = err
			results[i] = result
			continue
		}

		result.Success = true
		results[i] = result
	}

	return results, nil
}
