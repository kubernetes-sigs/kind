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
	"encoding/json"
	"io"
	"os"
	"path"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
)

// InspectArchive extracts metadata from docker save tar
//
// Supports multiple images in one tar (docker save can save multiple images)
// Returns metadata for ALL images found in the archive
func InspectArchive(tarPath string) (*ImageMetadata, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open tar")
	}
	defer f.Close()

	return inspectArchiveReader(f)
}

func inspectArchiveReader(r io.Reader) (*ImageMetadata, error) {
	tr := tar.NewReader(r)

	var manifestJSON []byte

	// Scan tar for manifest.json
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Name == "manifest.json" {
			manifestJSON, _ = io.ReadAll(tr)
			break
		}
	}

	if len(manifestJSON) == 0 {
		return nil, errors.New("manifest.json not found in archive")
	}

	return parseManifest(manifestJSON)
}

// dockerManifest is the Docker save format
// One entry per image
type dockerManifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

func parseManifest(manifestJSON []byte) (*ImageMetadata, error) {
	var manifests []dockerManifest
	if err := json.Unmarshal(manifestJSON, &manifests); err != nil {
		return nil, errors.Wrap(err, "failed to parse manifest.json")
	}

	if len(manifests) == 0 {
		return nil, errors.New("no images found in manifest")
	}

	meta := &ImageMetadata{
		Images: make([]ImageInfo, 0, len(manifests)),
	}

	// Parse each image
	for _, dm := range manifests {
		img := ImageInfo{
			RepoTags: dm.RepoTags,
		}

		// Extract config digest
		if strings.HasPrefix(dm.Config, "blobs/sha256/") {
			img.ConfigDigest = "sha256:" + path.Base(dm.Config)
		}

		// Extract layer digests
		for _, layerPath := range dm.Layers {
			digest := extractDigest(layerPath)
			img.Layers = append(img.Layers, LayerInfo{
				Digest:    digest,
				MediaType: "application/vnd.oci.image.layer.v1.tar",
			})
		}

		meta.Images = append(meta.Images, img)
	}

	return meta, nil
}

func extractDigest(p string) string {
	// Extract from: blobs/sha256/abc... or abc.tar
	if strings.HasPrefix(p, "blobs/sha256/") {
		return "sha256:" + path.Base(p)
	}
	base := path.Base(p)
	return "sha256:" + strings.TrimSuffix(base, ".tar")
}
