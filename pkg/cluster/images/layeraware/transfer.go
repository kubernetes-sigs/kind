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
	"io"

	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

// ExecuteTransfer executes a transfer plan.
//
// It streams a reduced archive to `ctr images import`: layer blobs already
// present in the node's content store are filtered out, and containerd sources
// them from the store during import. Only the layers missing from the node
// cross into it, so re-tagged or rebuilt images that share base layers with an
// image already on the node load without re-sending those layers.
func ExecuteTransfer(
	archivePath string,
	plan *TransferPlan,
	logger log.Logger,
) error {
	skip := skippableLayers(plan)

	if logger != nil {
		imageCount := plan.Metadata.ImageCount()
		if imageCount == 1 {
			logger.V(0).Infof("Loading image on node %s", plan.Node.String())
		} else {
			logger.V(0).Infof("Loading %d images on node %s", imageCount, plan.Node.String())
		}
	}

	// Filter the archive on the fly and pipe it straight into the node's
	// image importer. The pipe keeps memory constant for arbitrarily large
	// images; the writer goroutine reports what it skipped via reduceStats.
	pr, pw := io.Pipe()
	statsCh := make(chan reduceStats, 1)
	go func() {
		stats, err := writeReducedArchive(pw, archivePath, skip)
		statsCh <- stats
		// CloseWithError(nil) is equivalent to Close(): signals clean EOF.
		// The returned error is always nil (documented), so it is discarded.
		_ = pw.CloseWithError(err)
	}()

	importErr := nodeutils.LoadImageArchive(plan.Node, pr)
	// Drain the reader so the writer goroutine can never block on a failed
	// import, then collect its stats.
	_, _ = io.Copy(io.Discard, pr)
	stats := <-statsCh

	if importErr != nil {
		return errors.Wrap(importErr, "failed to import images")
	}

	logReduction(logger, plan.Node.String(), stats)
	return nil
}
