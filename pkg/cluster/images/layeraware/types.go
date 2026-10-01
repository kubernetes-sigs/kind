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
)

// ImageMetadata contains info from docker save archive
// Supports multiple images in one tar (like current kind)
type ImageMetadata struct {
	// All images in the tar
	Images []ImageInfo
}

// ImageInfo describes a single image
type ImageInfo struct {
	RepoTags     []string
	ConfigDigest string
	Layers       []LayerInfo
}

// LayerInfo describes a layer
type LayerInfo struct {
	Digest    string
	MediaType string
	// Note: Size not needed - we don't report it (current doesn't)
}

// GetAllDigests returns all unique blob digests across all images
func (m *ImageMetadata) GetAllDigests() []string {
	digestSet := make(map[string]bool)

	for _, img := range m.Images {
		if img.ConfigDigest != "" {
			digestSet[img.ConfigDigest] = true
		}
		for _, layer := range img.Layers {
			digestSet[layer.Digest] = true
		}
	}

	digests := make([]string, 0, len(digestSet))
	for d := range digestSet {
		digests = append(digests, d)
	}
	return digests
}

// TransferPlan describes what to transfer to a node.
//
// ExistingBlobs holds the digests (of the image's config + layers) already
// present in the node's content store; layer blobs in this set are filtered
// out of the archive during import.
type TransferPlan struct {
	Node          nodes.Node
	Metadata      *ImageMetadata
	ExistingBlobs map[string]bool
}

// ImageCount returns number of images
func (m *ImageMetadata) ImageCount() int {
	return len(m.Images)
}

// AllRepoTags returns all repo tags across all images
func (m *ImageMetadata) AllRepoTags() []string {
	var tags []string
	for _, img := range m.Images {
		tags = append(tags, img.RepoTags...)
	}
	return tags
}
