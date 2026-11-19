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

// LoadImageLayerAware loads an image archive onto a single node using
// layer-aware transfer: layers already present in the node's content store are
// not re-sent. metadata is obtained once via layeraware.InspectArchive and may
// be shared across nodes.
//
// Callers are responsible for falling back to a full load if this returns an
// error.
func LoadImageLayerAware(
	archivePath string,
	metadata *layeraware.ImageMetadata,
	node nodes.Node,
	logger log.Logger,
) error {
	plan, err := layeraware.PlanTransfer(metadata, node)
	if err != nil {
		return err
	}
	return layeraware.ExecuteTransfer(archivePath, plan, logger)
}
