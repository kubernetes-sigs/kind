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
	"reflect"
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
	linuxS390x    = `{"os":"linux","architecture":"s390x"}`
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

func TestCheckImageArchivePlatforms(t *testing.T) {
	t.Parallel()
	amd64, arm64, s390x := newTestBlob(`{"schemaVersion":2,"config":{"digest":"amd64"}}`), newTestBlob(`{"schemaVersion":2,"config":{"digest":"arm64"}}`), newTestBlob(`{"schemaVersion":2,"config":{"digest":"s390x"}}`)
	attestation := newTestBlob(`{"schemaVersion":2,"config":{"digest":"attestation"}}`)
	index := newTestBlob(testIndex(testDescriptor(ociManifest, amd64, linuxAmd64), testDescriptor(ociManifest, attestation, unknownTarget), testDescriptor(ociManifest, arm64, linuxArm64)))
	threePlatforms := newTestBlob(testIndex(testDescriptor(ociManifest, amd64, linuxAmd64), testDescriptor(ociManifest, arm64, linuxArm64), testDescriptor(ociManifest, s390x, linuxS390x)))
	s390xOnly := newTestBlob(testIndex(testDescriptor(ociManifest, s390x, linuxS390x)))
	list := newTestBlob(testIndex(testDescriptor(dockerImage, amd64, linuxAmd64), testDescriptor(dockerImage, arm64, linuxArm64)))
	self := testBlob{digest: "sha256:0123"}
	self.content = testIndex(testDescriptor(ociIndex, self, ""), testDescriptor(ociManifest, amd64, linuxAmd64))
	notJSON := newTestBlob("not json")
	full := testArchive(t, testIndex(testDescriptor(ociIndex, index, "")), index, amd64).Bytes()
	// cut inside the content of the last entry
	truncated := bytes.NewBuffer(full[:len(full)-1024-512+20])
	linuxUnknownIndex := newTestBlob(testIndex(testDescriptor(ociManifest, amd64, linuxAmd64), testDescriptor(ociManifest, attestation, `{"os":"linux","architecture":"unknown"}`), testDescriptor(ociManifest, arm64, linuxArm64)))

	cases := []struct {
		name     string
		archive  *bytes.Buffer
		included []string
		missing  []string
	}{
		{
			name:    "docker archive",
			archive: testTar(t, testFile{"manifest.json", "[]"}, amd64.file()),
		},
		{
			name:    "index.json without oci-layout",
			archive: testTar(t, testFile{"index.json", testIndex(testDescriptor(ociIndex, index, ""))}, index.file(), amd64.file()),
		},
		{
			name:    "index with all manifests",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, "")), index, amd64, attestation, arm64),
		},
		{
			name:     "index with platform manifests missing",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, index, "")), index, amd64),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8", "unknown/unknown"},
		},
		{
			name:     "several platforms included",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, threePlatforms, "")), threePlatforms, amd64, arm64),
			included: []string{"linux/amd64", "linux/arm64/v8"},
			missing:  []string{"linux/s390x"},
		},
		{
			name:     "only attestations missing",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, index, "")), index, amd64, arm64),
			included: []string{"linux/amd64", "linux/arm64/v8"},
			missing:  []string{"unknown/unknown"},
		},
		{
			name:     "docker manifest list with a platform manifest missing",
			archive:  testArchive(t, testIndex(testDescriptor(dockerList, list, "")), list, amd64),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8"},
		},
		{
			name:     "one image without an included platform",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociIndex, s390xOnly, "")), index, s390xOnly, amd64),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8", "linux/s390x", "unknown/unknown"},
		},
		{
			name:     "entries with ./ prefix",
			archive:  testTar(t, testFile{"./oci-layout", `{"imageLayoutVersion":"1.0.0"}`}, testFile{"./index.json", testIndex(testDescriptor(ociIndex, index, ""))}, testFile{"./" + index.file().name, index.content}, testFile{"./" + amd64.file().name, amd64.content}),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8", "unknown/unknown"},
		},
		{
			name:     "platform with only unknown architecture is not included",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, linuxUnknownIndex, "")), linuxUnknownIndex, amd64, attestation),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8"},
		},
		{
			name:    "truncated archive",
			archive: truncated,
		},
		{
			name:    "no platform included",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, s390xOnly, "")), s390xOnly),
		},
		{
			name:     "present attestation is not a platform",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, index, "")), index, amd64, attestation),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8"},
		},
		{
			name:     "index listed twice",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociIndex, index, "")), index, amd64),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8", "unknown/unknown"},
		},
		{
			name:    "index without platform missing",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociManifest, arm64, linuxArm64)), amd64),
		},
		{
			name:    "manifest without platform missing",
			archive: testArchive(t, testIndex(testDescriptor(ociManifest, amd64, linuxAmd64), testDescriptor(ociManifest, arm64, linuxArm64), testDescriptor(ociManifest, attestation, "")), amd64),
		},
		{
			name:    "index that is not JSON",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociIndex, notJSON, "")), index, amd64, notJSON),
		},
		{
			name:     "missing descriptor of another media type is ignored",
			archive:  testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor("application/vnd.oci.image.layer.v1.tar", newTestBlob("layer"), "")), index, amd64),
			included: []string{"linux/amd64"},
			missing:  []string{"linux/arm64/v8", "unknown/unknown"},
		},
		{
			name:    "same manifest missing with and without platform",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, index, ""), testDescriptor(ociManifest, attestation, "")), index, amd64),
		},
		{
			name:    "index listing itself",
			archive: testArchive(t, testIndex(testDescriptor(ociIndex, self, "")), self, amd64),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			included, missing := CheckImageArchivePlatforms(tc.archive)
			if !reflect.DeepEqual(included, tc.included) || !reflect.DeepEqual(missing, tc.missing) {
				t.Errorf("got %v and %v missing, want %v and %v missing", included, missing, tc.included, tc.missing)
			}
		})
	}
}
