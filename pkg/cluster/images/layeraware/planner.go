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

package layeraware

import (
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
)

// PlanTransfer creates a transfer plan for a node
//
// Computes which blobs need to be transferred across ALL images
func PlanTransfer(metadata *ImageMetadata, node nodes.Node) (*TransferPlan, error) {
	allDigests := metadata.GetAllDigests()

	// Query node
	existingMap, err := nodeutils.QueryContentBlobs(node, allDigests)
	if err != nil {
		return nil, err
	}

	// Compute plan: record which of the image's blobs already exist on the node.
	plan := &TransferPlan{
		Node:          node,
		Metadata:      metadata,
		ExistingBlobs: make(map[string]bool),
	}

	for _, digest := range allDigests {
		if existingMap[digest] {
			plan.ExistingBlobs[digest] = true
		}
	}

	return plan, nil
}
