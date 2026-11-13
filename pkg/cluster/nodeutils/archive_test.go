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
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

const (
	ociIndex      = "application/vnd.oci.image.index.v1+json"
	ociManifest   = "application/vnd.oci.image.manifest.v1+json"
	dockerList    = "application/vnd.docker.distribution.manifest.list.v2+json"
	dockerImage   = "application/vnd.docker.distribution.manifest.v2+json"
	linuxAmd64    = `{"os":"linux","architecture":"amd64"}`
	linuxArm64    = `{"os":"linux","architecture":"arm64","variant":"v8"}`
	unknownTarget = `{"os":"unknown","architecture":"unknown"}`
)

type testBlob struct {
	digest  string
	content string
}

func newTestBlob(content string) testBlob {
	return testBlob{digest: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content))), content: content}
}

// testDescriptor omits the platform when empty.
func testDescriptor(mediaType string, b testBlob, platform string) string {
	if platform == "" {
		return fmt.Sprintf(`{"mediaType":%q,"digest":%q}`, mediaType, b.digest)
	}
	return fmt.Sprintf(`{"mediaType":%q,"digest":%q,"platform":%s}`, mediaType, b.digest, platform)
}

func testIndex(descriptors ...string) string {
	return fmt.Sprintf(`{"schemaVersion":2,"manifests":[%s]}`, strings.Join(descriptors, ","))
}

func (b testBlob) file() testFile {
	return testFile{"blobs/sha256/" + strings.TrimPrefix(b.digest, "sha256:"), b.content}
}

type testFile struct{ name, content string }

func testTar(t *testing.T, files ...testFile) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, f := range files {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

func testArchive(t *testing.T, index string, blobs ...testBlob) *bytes.Buffer {
	t.Helper()
	files := []testFile{{"oci-layout", `{"imageLayoutVersion":"1.0.0"}`}, {"index.json", index}}
	for _, b := range blobs {
		files = append(files, b.file())
	}
	return testTar(t, files...)
}

func TestMissingPlatformManifests(t *testing.T) {
	t.Parallel()
	amd64, arm64 := newTestBlob(`{"schemaVersion":2,"config":{"digest":"amd64"}}`), newTestBlob(`{"schemaVersion":2,"config":{"digest":"arm64"}}`)
	attestation := newTestBlob(`{"schemaVersion":2,"config":{"digest":"attestation"}}`)
	index := newTestBlob(testIndex(testDescriptor(ociManifest, amd64, linuxAmd64), testDescriptor(ociManifest, attestation, unknownTarget), testDescriptor(ociManifest, arm64, linuxArm64)))
	list := newTestBlob(testIndex(testDescriptor(dockerImage, amd64, linuxAmd64), testDescriptor(dockerImage, arm64, linuxArm64)))
	self := testBlob{digest: "sha256:0123"}
	self.content = testIndex(testDescriptor(ociIndex, self, ""), testDescriptor(ociManifest, amd64, linuxAmd64))
	notJSON := newTestBlob("not json")

	cases := []struct {
		name    string
		archive *bytes.Buffer
		want    bool
	}{
		{
			name:    "docker archive",
			archive: testTar(t, testFile{"manifest.json", "[]"}, amd64.file()),
			want:    false,
		},
		{
			name:    "index.json without oci-layout",
			archive: testTar(t, testFile{"index.json", testIndex(testDescriptor(ociIndex, index, ""))}, index.file(), amd64.file()),
			want:    false,
		},
		{
			name:    "index with all manifests",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, "")), index, amd64, attestation, arm64),
			want:    false,
		},
		{
			name:    "index with platform manifests missing",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, "")), index, amd64),
			want:    true,
		},
		{
			name:    "docker manifest list with a platform manifest missing",
			archive: testArchive(t, testIndex(testDescriptor(dockerList, list, "")), list, amd64),
			want:    true,
		},
		{
			name:    "index without platform missing",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociManifest, arm64, linuxArm64)), amd64),
			want:    false,
		},
		{
			name:    "manifest without platform missing",
			archive: testArchive(t, testIndex(testDescriptor(ociManifest, amd64, linuxAmd64), testDescriptor(ociManifest, arm64, linuxArm64), testDescriptor(ociManifest, attestation, "")), amd64),
			want:    false,
		},
		{
			name:    "index that is not JSON",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociIndex, notJSON, "")), index, amd64, notJSON),
			want:    false,
		},
		{
			name:    "missing descriptor of another media type is ignored",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor("application/vnd.oci.image.layer.v1.tar", newTestBlob("layer"), "")), index, amd64),
			want:    true,
		},
		{
			name:    "same manifest missing with and without platform",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociManifest, attestation, "")), index, amd64),
			want:    false,
		},
		{
			name:    "index listing itself",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, self, "")), self, amd64),
			want:    false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := missingPlatformManifests(tc.archive); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
