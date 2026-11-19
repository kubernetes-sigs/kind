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
	"bytes"
	"errors"
	"testing"
)

func TestParseManifest_SingleImage(t *testing.T) {
	manifestJSON := []byte(`[
		{
			"Config": "blobs/sha256/abc123.json",
			"RepoTags": ["nginx:latest"],
			"Layers": [
				"blobs/sha256/layer1.tar",
				"blobs/sha256/layer2.tar"
			]
		}
	]`)

	metadata, err := parseManifest(manifestJSON)
	if err != nil {
		t.Fatalf("parseManifest() error = %v", err)
	}

	if len(metadata.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(metadata.Images))
	}

	img := metadata.Images[0]
	if len(img.RepoTags) != 1 || img.RepoTags[0] != "nginx:latest" {
		t.Errorf("expected RepoTags [nginx:latest], got %v", img.RepoTags)
	}

	if img.ConfigDigest != "sha256:abc123.json" {
		t.Errorf("expected ConfigDigest sha256:abc123.json, got %s", img.ConfigDigest)
	}

	if len(img.Layers) != 2 {
		t.Errorf("expected 2 layers, got %d", len(img.Layers))
	}
}

func TestParseManifest_MultiImage(t *testing.T) {
	manifestJSON := []byte(`[
		{
			"Config": "blobs/sha256/config1.json",
			"RepoTags": ["nginx:latest", "nginx:1.0"],
			"Layers": ["blobs/sha256/layer1.tar"]
		},
		{
			"Config": "blobs/sha256/config2.json",
			"RepoTags": ["redis:alpine"],
			"Layers": ["blobs/sha256/layer2.tar", "blobs/sha256/layer3.tar"]
		}
	]`)

	metadata, err := parseManifest(manifestJSON)
	if err != nil {
		t.Fatalf("parseManifest() error = %v", err)
	}

	if len(metadata.Images) != 2 {
		t.Errorf("expected 2 images, got %d", len(metadata.Images))
	}

	// Check first image
	if len(metadata.Images[0].RepoTags) != 2 {
		t.Errorf("expected 2 tags for first image, got %d", len(metadata.Images[0].RepoTags))
	}

	// Check second image
	if len(metadata.Images[1].Layers) != 2 {
		t.Errorf("expected 2 layers for second image, got %d", len(metadata.Images[1].Layers))
	}
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	manifestJSON := []byte(`invalid json`)

	_, err := parseManifest(manifestJSON)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseManifest_EmptyArray(t *testing.T) {
	manifestJSON := []byte(`[]`)

	_, err := parseManifest(manifestJSON)
	if err == nil {
		t.Error("expected error for empty manifest, got nil")
	}
}

func TestBlobDigest(t *testing.T) {
	// OCI-layout blob paths are accepted and converted to sha256:<base>.
	if got, err := blobDigest("blobs/sha256/abc123def456"); err != nil || got != "sha256:abc123def456" {
		t.Errorf("blobDigest(OCI path) = %q, %v; want %q, nil", got, err, "sha256:abc123def456")
	}

	// Non-OCI paths (legacy docker save, or malformed) are rejected so the
	// caller falls back to a full load.
	for _, p := range []string{
		"abc123.tar",       // legacy layer tar at root
		"layer.tar",        // plain filename
		"a1b2c3/layer.tar", // legacy "<id>/layer.tar"
		"a1b2c3.json",      // legacy config
		"blobs/sha256/",    // empty digest
	} {
		if _, err := blobDigest(p); !errors.Is(err, ErrUnsupportedArchive) {
			t.Errorf("blobDigest(%q) err = %v; want ErrUnsupportedArchive", p, err)
		}
	}
}

func TestParseManifest_LegacyFormatUnsupported(t *testing.T) {
	// Legacy docker save: config is "<id>.json", layers are "<id>/layer.tar",
	// with no content-addressed blobs. Must be rejected (ErrUnsupportedArchive)
	// so the caller falls back to a full load.
	manifestJSON := []byte(`[
		{
			"Config": "a1b2c3.json",
			"RepoTags": ["nginx:latest"],
			"Layers": ["a1b2c3/layer.tar", "d4e5f6/layer.tar"]
		}
	]`)

	if _, err := parseManifest(manifestJSON); !errors.Is(err, ErrUnsupportedArchive) {
		t.Fatalf("parseManifest(legacy) err = %v; want ErrUnsupportedArchive", err)
	}
}

func TestInspectArchiveReader_Success(t *testing.T) {
	// Create a minimal tar archive with manifest.json
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	manifestJSON := []byte(`[
		{
			"Config": "blobs/sha256/config.json",
			"RepoTags": ["test:latest"],
			"Layers": ["blobs/sha256/layer1.tar"]
		}
	]`)

	// Write manifest.json to tar
	err := tw.WriteHeader(&tar.Header{
		Name: "manifest.json",
		Mode: 0644,
		Size: int64(len(manifestJSON)),
	})
	if err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}

	_, err = tw.Write(manifestJSON)
	if err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	err = tw.Close()
	if err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	// Test inspectArchiveReader
	metadata, err := inspectArchiveReader(&buf)
	if err != nil {
		t.Fatalf("inspectArchiveReader() error = %v", err)
	}

	if len(metadata.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(metadata.Images))
	}

	if metadata.Images[0].RepoTags[0] != "test:latest" {
		t.Errorf("expected RepoTags [test:latest], got %v", metadata.Images[0].RepoTags)
	}
}

func TestInspectArchiveReader_NoManifest(t *testing.T) {
	// Create a tar archive without manifest.json
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Write some other file
	err := tw.WriteHeader(&tar.Header{
		Name: "other.txt",
		Mode: 0644,
		Size: 4,
	})
	if err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}

	_, err = tw.Write([]byte("test"))
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}

	err = tw.Close()
	if err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	// Test inspectArchiveReader
	_, err = inspectArchiveReader(&buf)
	if err == nil {
		t.Error("expected error for missing manifest.json, got nil")
	}
}

func TestInspectArchiveReader_MultipleImages(t *testing.T) {
	// Create a tar archive with multi-image manifest
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	manifestJSON := []byte(`[
		{
			"Config": "blobs/sha256/config1.json",
			"RepoTags": ["nginx:latest"],
			"Layers": ["blobs/sha256/layer1.tar"]
		},
		{
			"Config": "blobs/sha256/config2.json",
			"RepoTags": ["redis:alpine"],
			"Layers": ["blobs/sha256/layer2.tar"]
		}
	]`)

	err := tw.WriteHeader(&tar.Header{
		Name: "manifest.json",
		Mode: 0644,
		Size: int64(len(manifestJSON)),
	})
	if err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}

	_, err = tw.Write(manifestJSON)
	if err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	err = tw.Close()
	if err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	// Test inspectArchiveReader
	metadata, err := inspectArchiveReader(&buf)
	if err != nil {
		t.Fatalf("inspectArchiveReader() error = %v", err)
	}

	if len(metadata.Images) != 2 {
		t.Errorf("expected 2 images, got %d", len(metadata.Images))
	}

	// Verify GetAllDigests works with multiple images
	digests := metadata.GetAllDigests()
	if len(digests) != 4 { // 2 configs + 2 layers
		t.Errorf("expected 4 unique digests, got %d", len(digests))
	}
}
