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
	"testing"
)

func TestImageMetadata_GetAllDigests(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ImageMetadata
		want     int // number of unique digests
	}{
		{
			name: "single image with layers",
			metadata: &ImageMetadata{
				Images: []ImageInfo{
					{
						ConfigDigest: "sha256:config1",
						Layers: []LayerInfo{
							{Digest: "sha256:layer1"},
							{Digest: "sha256:layer2"},
						},
					},
				},
			},
			want: 3, // config + 2 layers
		},
		{
			name: "multiple images with shared layers",
			metadata: &ImageMetadata{
				Images: []ImageInfo{
					{
						ConfigDigest: "sha256:config1",
						Layers: []LayerInfo{
							{Digest: "sha256:layer1"},
							{Digest: "sha256:layer2"},
						},
					},
					{
						ConfigDigest: "sha256:config2",
						Layers: []LayerInfo{
							{Digest: "sha256:layer1"}, // shared
							{Digest: "sha256:layer3"},
						},
					},
				},
			},
			want: 5, // 2 configs + 3 unique layers
		},
		{
			name: "image without config digest",
			metadata: &ImageMetadata{
				Images: []ImageInfo{
					{
						Layers: []LayerInfo{
							{Digest: "sha256:layer1"},
						},
					},
				},
			},
			want: 1, // just the layer
		},
		{
			name: "empty metadata",
			metadata: &ImageMetadata{
				Images: []ImageInfo{},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.GetAllDigests()
			if len(got) != tt.want {
				t.Errorf("GetAllDigests() returned %d digests, want %d", len(got), tt.want)
			}

			// Check for duplicates
			seen := make(map[string]bool)
			for _, digest := range got {
				if seen[digest] {
					t.Errorf("GetAllDigests() returned duplicate digest: %s", digest)
				}
				seen[digest] = true
			}
		})
	}
}

func TestImageMetadata_ImageCount(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ImageMetadata
		want     int
	}{
		{
			name: "single image",
			metadata: &ImageMetadata{
				Images: []ImageInfo{{}},
			},
			want: 1,
		},
		{
			name: "multiple images",
			metadata: &ImageMetadata{
				Images: []ImageInfo{{}, {}, {}},
			},
			want: 3,
		},
		{
			name: "no images",
			metadata: &ImageMetadata{
				Images: []ImageInfo{},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metadata.ImageCount(); got != tt.want {
				t.Errorf("ImageCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImageMetadata_AllRepoTags(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ImageMetadata
		want     []string
	}{
		{
			name: "single image with tags",
			metadata: &ImageMetadata{
				Images: []ImageInfo{
					{RepoTags: []string{"nginx:latest", "nginx:1.0"}},
				},
			},
			want: []string{"nginx:latest", "nginx:1.0"},
		},
		{
			name: "multiple images",
			metadata: &ImageMetadata{
				Images: []ImageInfo{
					{RepoTags: []string{"nginx:latest"}},
					{RepoTags: []string{"redis:alpine", "redis:7"}},
				},
			},
			want: []string{"nginx:latest", "redis:alpine", "redis:7"},
		},
		{
			name: "no tags",
			metadata: &ImageMetadata{
				Images: []ImageInfo{
					{RepoTags: []string{}},
				},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.AllRepoTags()
			if len(got) != len(tt.want) {
				t.Errorf("AllRepoTags() returned %d tags, want %d", len(got), len(tt.want))
			}
			for i, tag := range got {
				if tag != tt.want[i] {
					t.Errorf("AllRepoTags()[%d] = %v, want %v", i, tag, tt.want[i])
				}
			}
		})
	}
}
