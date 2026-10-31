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
	"io"
	"os"
	"path/filepath"
	"testing"
)

// writeTar writes a tar with the given name->content entries to path.
func writeTar(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	for name, content := range entries {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0o644,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
}

// readTarNames returns the set of entry names in a tar stream.
func readTarNames(t *testing.T, r io.Reader) map[string]string {
	t.Helper()
	out := map[string]string{}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(tr)
		out[hdr.Name] = string(b)
	}
	return out
}

func TestWriteReducedArchive(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "image.tar")

	present := "blobs/sha256/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	missing := "blobs/sha256/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	config := "blobs/sha256/cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	writeTar(t, src, map[string]string{
		"manifest.json": `[{"Config":"` + config + `"}]`,
		"index.json":    `{"schemaVersion":2}`,
		present:         "PRESENT-LAYER-DATA",
		missing:         "MISSING-LAYER-DATA",
		config:          "CONFIG-DATA",
	})

	// Mark the "present" layer as skippable (already on the node).
	skip := map[string]bool{"sha256:" + filepath.Base(present): true}

	var buf bytes.Buffer
	stats, err := writeReducedArchive(&buf, src, skip)
	if err != nil {
		t.Fatalf("writeReducedArchive: %v", err)
	}

	got := readTarNames(t, &buf)

	// The present layer must be dropped; everything else must survive.
	if _, ok := got[present]; ok {
		t.Errorf("present layer should have been skipped, but it is in the reduced archive")
	}
	for _, want := range []string{"manifest.json", "index.json", missing, config} {
		if _, ok := got[want]; !ok {
			t.Errorf("reduced archive is missing required entry %q", want)
		}
	}

	if stats.SkippedLayers != 1 {
		t.Errorf("SkippedLayers = %d, want 1", stats.SkippedLayers)
	}
	if want := int64(len("PRESENT-LAYER-DATA")); stats.SkippedBytes != want {
		t.Errorf("SkippedBytes = %d, want %d", stats.SkippedBytes, want)
	}
}

func TestDigestFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "blob path", path: "blobs/sha256/abc123", want: "abc123"},
		{name: "tar file", path: "layer.tar", want: "layer"},
		{name: "nested blob path", path: "some/dir/blobs/sha256/digest456", want: "digest456"},
		{name: "file with .tar extension", path: "abc123.tar", want: "abc123"},
		{name: "path without .tar", path: "somefile", want: "somefile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := digestFromPath(tt.path); got != tt.want {
				t.Errorf("digestFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestSkippableLayersOnlyLayers(t *testing.T) {
	// Config digest must never be marked skippable, even if it is "present".
	configDigest := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	layerDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	plan := &TransferPlan{
		Metadata: &ImageMetadata{
			Images: []ImageInfo{{
				ConfigDigest: configDigest,
				Layers:       []LayerInfo{{Digest: layerDigest}},
			}},
		},
		ExistingBlobs: map[string]bool{
			configDigest: true,
			layerDigest:  true,
		},
	}

	skip := skippableLayers(plan)
	if !skip[layerDigest] {
		t.Errorf("present layer should be skippable")
	}
	if skip[configDigest] {
		t.Errorf("config blob must never be skippable")
	}
}
