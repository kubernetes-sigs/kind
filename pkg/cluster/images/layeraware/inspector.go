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

// ErrUnsupportedArchive indicates the image archive is not an OCI-layout
// `docker save` (content-addressed blobs/sha256/... entries), so layer-aware
// loading cannot process it. Callers should fall back to a full load.
//
// It is returned unwrapped so callers can match it with errors.Is.
var ErrUnsupportedArchive = errors.New("unsupported image archive format for layer-aware load")

// InspectArchive extracts metadata from docker save tar
//
// Supports multiple images in one tar (docker save can save multiple images).
// Returns metadata for ALL images found in the archive, or ErrUnsupportedArchive
// if the archive is not an OCI layout that layer-aware loading can handle.
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

	// Parse each image.
	//
	// Layer-aware loading only supports the OCI-layout docker save format, where
	// the config and every layer are content-addressed blobs
	// (blobs/sha256/<digest>). The legacy format references layers by filename
	// with no digest and cannot be skipped (containerd's importer hashes each
	// layer from the tar and fails if a referenced file is absent, even when the
	// blob already exists in the content store). Reject anything else so the
	// caller falls back to a full load.
	for _, dm := range manifests {
		configDigest, err := blobDigest(dm.Config)
		if err != nil {
			return nil, err
		}
		img := ImageInfo{
			RepoTags:     dm.RepoTags,
			ConfigDigest: configDigest,
		}
		for _, layerPath := range dm.Layers {
			digest, err := blobDigest(layerPath)
			if err != nil {
				return nil, err
			}
			img.Layers = append(img.Layers, LayerInfo{Digest: digest})
		}
		meta.Images = append(meta.Images, img)
	}

	return meta, nil
}

// blobDigest converts an OCI-layout blob path ("blobs/sha256/<hex>") into a
// "sha256:<hex>" digest. Any other shape (such as the legacy "<id>/layer.tar"
// or "<id>.json") yields ErrUnsupportedArchive.
func blobDigest(p string) (string, error) {
	if !strings.HasPrefix(p, "blobs/sha256/") {
		return "", ErrUnsupportedArchive
	}
	hex := path.Base(p)
	if hex == "" || hex == "sha256" {
		return "", ErrUnsupportedArchive
	}
	return "sha256:" + hex, nil
}
