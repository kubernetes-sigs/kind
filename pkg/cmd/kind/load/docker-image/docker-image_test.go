/*
Copyright 2021 The Kubernetes Authors.

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

package load

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

func Test_removeDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		want  []string
	}{
		{
			name:  "empty",
			slice: []string{},
			want:  []string{},
		},
		{
			name:  "all different",
			slice: []string{"one", "two"},
			want:  []string{"one", "two"},
		},
		{
			name:  "one dup",
			slice: []string{"one", "two", "two"},
			want:  []string{"one", "two"},
		},
		{
			name:  "two dup",
			slice: []string{"one", "two", "two", "one"},
			want:  []string{"one", "two"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeDuplicates(tt.slice)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeDuplicates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sanitizeImage(t *testing.T) {
	tests := []struct {
		name           string
		image          string
		sanitizedImage string
	}{
		{
			image:          "ubuntu:18.04",
			sanitizedImage: "docker.io/library/ubuntu:18.04",
		},
		{
			image:          "custom/ubuntu:18.04",
			sanitizedImage: "docker.io/custom/ubuntu:18.04",
		},
		{
			image:          "registry.k8s.io/kindest/node:latest",
			sanitizedImage: "registry.k8s.io/kindest/node:latest",
		},
		{
			image:          "registry.k8s.io/pause:3.6",
			sanitizedImage: "registry.k8s.io/pause:3.6",
		},
		{
			image:          "baz",
			sanitizedImage: "docker.io/library/baz:latest",
		},
		{
			image:          "other-registry/baz",
			sanitizedImage: "docker.io/other-registry/baz:latest",
		},
		{
			image:          "localhost:5000/baz",
			sanitizedImage: "localhost:5000/baz:latest",
		},
		{
			image:          "localhost:5000/baz:quux",
			sanitizedImage: "localhost:5000/baz:quux",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeImage(tt.image)
			if got != tt.sanitizedImage {
				t.Errorf("sanitizeImage(%s) = %s, want %s", tt.image, got, tt.sanitizedImage)
			}
		})
	}
}

func Test_checkIfImageReTagRequired(t *testing.T) {
	tests := []struct {
		name      string
		imageTags struct {
			tags map[string]bool
			err  error
		}
		imageID        string
		imageName      string
		returnValues   []bool
		sanitizedImage string
	}{
		{
			name: "image is already present",
			imageTags: struct {
				tags map[string]bool
				err  error
			}{
				map[string]bool{
					"docker.io/library/image1:tag1": true,
					"k8s.io/image1:tag1":            true,
				},
				nil,
			},
			imageID:        "sha256:fd3fd9ab134a864eeb7b2c073c0d90192546f597c60416b81fc4166cca47f29a",
			imageName:      "k8s.io/image1:tag1",
			returnValues:   []bool{true, false},
			sanitizedImage: "k8s.io/image1:tag1",
		},
		{
			name: "re-tag is required",
			imageTags: struct {
				tags map[string]bool
				err  error
			}{
				map[string]bool{
					"docker.io/library/image1:tag1": true,
					"k8s.io/image1:tag1":            true,
				},
				nil,
			},
			imageID:        "sha256:fd3fd9ab134a864eeb7b2c073c0d90192546f597c60416b81fc4166cca47f29a",
			imageName:      "k8s.io/image1:tag2",
			returnValues:   []bool{true, true},
			sanitizedImage: "k8s.io/image1:tag2",
		},
		{
			name: "re-tag is required with docker.io prefix",
			imageTags: struct {
				tags map[string]bool
				err  error
			}{
				map[string]bool{
					"docker.io/foo/image1:tag1": true,
				},
				nil,
			},
			imageID:        "sha256:fd3fd9ab134a864eeb7b2c073c0d90192546f597c60416b81fc4166cca47f29a",
			imageName:      "foo/image1:tag2",
			returnValues:   []bool{true, true},
			sanitizedImage: "docker.io/foo/image1:tag2",
		},
		{
			name: "image tag fetch failed",
			imageTags: struct {
				tags map[string]bool
				err  error
			}{
				map[string]bool{},
				errors.New("some runtime error"),
			},
			imageID:        "sha256:fd3fd9ab134a864eeb7b2c073c0d90192546f597c60416b81fc4166cca47f29a",
			imageName:      "k8s.io/image1:tag2",
			returnValues:   []bool{false, false},
			sanitizedImage: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// checkIfImageReTagRequired doesn't use the `nodes.Node` type for anything. So
			// passing a nil value here should be fine as the other two functions that use the
			// nodes.Node has been stubbed out already
			exists, reTagRequired, sanitizedImage := checkIfImageReTagRequired(nil, tc.imageID, tc.imageName, func(n nodes.Node, s string) (map[string]bool, error) {
				return tc.imageTags.tags, tc.imageTags.err
			})
			if exists != tc.returnValues[0] || reTagRequired != tc.returnValues[1] || sanitizedImage != tc.sanitizedImage {
				t.Errorf("checkIfImageReTagRequired failed. Expected: [%v,%v,%v], got: [%v, %v, %v]", tc.returnValues[0], tc.returnValues[1], tc.sanitizedImage, exists, reTagRequired, sanitizedImage)
			}
		})
	}
}

func Test_imagePresentOnNode(t *testing.T) {
	const (
		configDigest = "sha256:5a13d892fcdce272129e6e395d8fe5baa9532e61569c5a150e600589e81fd0a8"
		targetDigest = "sha256:b8d31d6ecf8e6a63a9cfa29c13e45e3c00a0d7fef86f59457aa8d76a812a461f"
	)
	notPresent := errors.New("no such image")

	tests := []struct {
		name          string
		imageID       string
		nodeID        string
		nodeIDErr     error
		nodeDigest    string
		nodeDigestErr error
		want          bool
	}{
		{
			// docker with the classic image store reports the config digest
			name:       "config digest matches",
			imageID:    configDigest,
			nodeID:     configDigest,
			nodeDigest: targetDigest,
			want:       true,
		},
		{
			// docker with the containerd image store reports the target digest
			name:       "target digest matches",
			imageID:    targetDigest,
			nodeID:     configDigest,
			nodeDigest: targetDigest,
			want:       true,
		},
		{
			// the tag exists on the node but points at different content
			name:       "image on node is stale",
			imageID:    "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			nodeID:     configDigest,
			nodeDigest: targetDigest,
			want:       false,
		},
		{
			name:          "image not present on node",
			imageID:       targetDigest,
			nodeIDErr:     notPresent,
			nodeDigestErr: notPresent,
			want:          false,
		},
		{
			// CRI cannot resolve a target digest, the containerd lookup still can
			name:       "cri lookup fails but target digest matches",
			imageID:    targetDigest,
			nodeIDErr:  notPresent,
			nodeDigest: targetDigest,
			want:       true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// imagePresentOnNode only passes nodes.Node through to the fetchers,
			// which are stubbed out here, so a nil node is fine
			got := imagePresentOnNode(nil, "image1:tag1", tc.imageID,
				func(nodes.Node, string) (string, error) {
					return tc.nodeID, tc.nodeIDErr
				},
				func(nodes.Node, string) (string, error) {
					return tc.nodeDigest, tc.nodeDigestErr
				},
			)
			if got != tc.want {
				t.Errorf("imagePresentOnNode() = %v, want %v", got, tc.want)
			}
		})
	}
}
