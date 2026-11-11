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

package nodeutils

import (
	"archive/tar"
	"encoding/json"
	"io"
	"path"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

const (
	// registries reject larger manifests
	maxJSONManifestSize = 4 << 20
)

// LoadImageArchiveSkipMissing is LoadImageArchive that loads only the node platform from archives missing some platforms, and reports whether it did.
func LoadImageArchiveSkipMissing(n nodes.Node, image io.ReadSeeker) (bool, error) {
	start, err := image.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, loadImageArchive(n, image, true)
	}
	partial := missingPlatformManifests(image)
	if _, err := image.Seek(start, io.SeekStart); err != nil {
		return false, errors.Wrap(err, "failed to rewind image archive")
	}
	if !partial {
		return false, loadImageArchive(n, image, true)
	}
	return true, loadImageArchive(n, image, false)
}

type descriptor struct {
	MediaType string    `json:"mediaType"`
	Digest    string    `json:"digest"`
	Platform  *struct{} `json:"platform"`
}

// missingPlatformManifests reports whether some index or manifest blobs are missing, all of them with a platform; unreadable archives report false.
func missingPlatformManifests(r io.Reader) bool {
	var (
		present = map[string]bool{}
		indexes = map[string][]descriptor{}
		tr      = tar.NewReader(r)
	)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		name := path.Clean(hdr.Name)
		present[name] = true
		delete(indexes, name)
		if hdr.Size <= maxJSONManifestSize {
			if manifests, ok := readIndex(tr); ok {
				indexes[name] = manifests
			}
		}
	}

	top, ok := indexes["index.json"]
	if !present["oci-layout"] || !ok {
		return false
	}

	w := archiveWalker{present: present, indexes: indexes, seen: map[string]bool{}}
	if err := w.walkIndex(top); err != nil {
		return false
	}

	return w.missingWithPlatform && !w.missingWithoutPlatform
}

// readIndex returns the manifests of r if it is an image index.
func readIndex(r io.Reader) ([]descriptor, bool) {
	var idx struct {
		Manifests []descriptor `json:"manifests"`
	}
	if err := json.NewDecoder(r).Decode(&idx); err != nil || idx.Manifests == nil {
		return nil, false
	}
	return idx.Manifests, true
}

type archiveWalker struct {
	present                map[string]bool
	indexes                map[string][]descriptor
	seen                   map[string]bool
	missingWithPlatform    bool
	missingWithoutPlatform bool
}

func (w *archiveWalker) walkIndex(manifests []descriptor) error {
	for _, d := range manifests {
		if err := w.walk(d); err != nil {
			return err
		}
	}
	return nil
}

func (w *archiveWalker) walk(d descriptor) error {
	isIndex := d.MediaType == "application/vnd.oci.image.index.v1+json" || d.MediaType == "application/vnd.docker.distribution.manifest.list.v2+json"
	isManifest := d.MediaType == "application/vnd.oci.image.manifest.v1+json" || d.MediaType == "application/vnd.docker.distribution.manifest.v2+json"
	if !isIndex && !isManifest {
		return nil
	}
	if !w.present[blobPath(d.Digest)] {
		// importing one platform still reads descriptors without a platform
		if d.Platform == nil {
			w.missingWithoutPlatform = true
		} else {
			w.missingWithPlatform = true
		}
		return nil
	}
	if !isIndex || w.seen[d.Digest] {
		return nil
	}
	w.seen[d.Digest] = true
	manifests, ok := w.indexes[blobPath(d.Digest)]
	if !ok {
		return errors.Errorf("failed to parse image index %s", d.Digest)
	}
	return w.walkIndex(manifests)
}

// blobPath returns the path of a blob in an OCI image layout
func blobPath(digest string) string {
	return "blobs/" + strings.Replace(digest, ":", "/", 1)
}
