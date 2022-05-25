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
			image:          "k8s.gcr.io/pause:3.6",
			sanitizedImage: "k8s.gcr.io/pause:3.6",
		},
		{
			image:          "baz",
			sanitizedImage: "docker.io/library/baz:latest",
		},
		{
			image:          "other-registry/baz",
			sanitizedImage: "docker.io/other-registry/baz:latest",
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
		imageID      string
		imageName    string
		returnValues []bool
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
			imageID:      "sha256:fd3fd9ab134a864eeb7b2c073c0d90192546f597c60416b81fc4166cca47f29a",
			imageName:    "k8s.io/image1:tag1",
			returnValues: []bool{true, false},
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
			imageID:      "sha256:fd3fd9ab134a864eeb7b2c073c0d90192546f597c60416b81fc4166cca47f29a",
			imageName:    "k8s.io/image1:tag2",
			returnValues: []bool{true, true},
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
			imageID:      "sha256:fd3fd9ab134a864eeb7b2c073c0d90192546f597c60416b81fc4166cca47f29a",
			imageName:    "k8s.io/image1:tag2",
			returnValues: []bool{false, false},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// checkIfImageReTagRequired doesn't use the `nodes.Node` type for anything. So
			// passing a nil value here should be fine as the other two functions that use the
			// nodes.Node has been stubbed out already
			exists, reTagRequired := checkIfImageReTagRequired(nil, tc.imageID, tc.imageName, func(n nodes.Node, s string) (map[string]bool, error) {
				return tc.imageTags.tags, tc.imageTags.err
			})
			if exists != tc.returnValues[0] || reTagRequired != tc.returnValues[1] {
				t.Errorf("checkIfImageReTagRequired failed. Expected: %v, got: [%v, %v]", tc.returnValues, exists, reTagRequired)
			}
		})
	}
}
