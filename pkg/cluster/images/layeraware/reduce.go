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
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"
)

// digestFromPath extracts a blob digest (hex, no algorithm prefix) from a tar
// entry name such as "blobs/sha256/<hex>" or "<hex>.tar". Non-blob entries
// (manifest.json, index.json, ...) yield their base name, which never matches a
// digest and is therefore always kept by the archive filter.
func digestFromPath(p string) string {
	if strings.HasPrefix(p, "blobs/sha256/") {
		return path.Base(p)
	}
	base := path.Base(p)
	return strings.TrimSuffix(base, ".tar")
}

// skippableLayers returns the set of layer blob digests (normalized to
// "sha256:<hex>") that already exist in the node's content store and can
// therefore be omitted from the archive we hand to `ctr images import`.
//
// Only *layer* blobs are ever skipped. Config and manifest blobs are always
// included: they are tiny, and importing an archive without its config blob is
// not reliably supported.
func skippableLayers(plan *TransferPlan) map[string]bool {
	skip := make(map[string]bool)
	for _, img := range plan.Metadata.Images {
		for _, layer := range img.Layers {
			if plan.ExistingBlobs[layer.Digest] {
				skip[layer.Digest] = true
			}
		}
	}
	return skip
}

// reduceStats reports what a reduced archive left out.
type reduceStats struct {
	SkippedLayers int
	SentBytes     int64
	SkippedBytes  int64
}

// writeReducedArchive copies the docker-save tar at srcPath to dst, dropping
// any blob entry whose digest is in skip (layers already present on the node).
// containerd sources the omitted layers from its content store during import,
// so the resulting image is complete without re-sending those bytes.
//
// The copy is single-pass and streaming: memory stays constant regardless of
// image size.
func writeReducedArchive(dst io.Writer, srcPath string, skip map[string]bool) (reduceStats, error) {
	var stats reduceStats

	f, err := os.Open(srcPath)
	if err != nil {
		return stats, errors.Wrap(err, "failed to open archive")
	}
	defer f.Close()

	tr := tar.NewReader(f)
	tw := tar.NewWriter(dst)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, errors.Wrap(err, "failed to read archive")
		}

		digest := digestFromPath(hdr.Name)
		if digest != "" {
			normalized := digest
			if !strings.HasPrefix(normalized, "sha256:") {
				normalized = "sha256:" + normalized
			}
			if skip[normalized] {
				// Layer already on the node: drop it from the archive.
				if _, err := io.Copy(io.Discard, tr); err != nil {
					return stats, errors.Wrap(err, "failed to skip blob")
				}
				stats.SkippedLayers++
				stats.SkippedBytes += hdr.Size
				continue
			}
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return stats, errors.Wrap(err, "failed to write archive header")
		}
		n, err := io.Copy(tw, tr)
		if err != nil {
			return stats, errors.Wrap(err, "failed to copy archive entry")
		}
		stats.SentBytes += n
	}

	if err := tw.Close(); err != nil {
		return stats, errors.Wrap(err, "failed to finalize archive")
	}
	return stats, nil
}

// logReduction emits a human-readable summary of what layer-aware skipped.
func logReduction(logger log.Logger, node string, stats reduceStats) {
	if logger == nil {
		return
	}
	if stats.SkippedLayers == 0 {
		logger.V(0).Infof("No existing layers to skip on node %s; sending full image", node)
		return
	}
	logger.V(0).Infof(
		"Skipped %d existing layer(s) on node %s: sent %s, skipped %s",
		stats.SkippedLayers, node,
		humanBytes(stats.SentBytes), humanBytes(stats.SkippedBytes),
	)
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
