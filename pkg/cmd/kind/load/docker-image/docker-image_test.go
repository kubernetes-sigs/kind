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
	"context"
	"errors"
	"io"
	"reflect"
	"sort"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// fakeNode is a nodes.Node that only implements String()
type fakeNode string

func (f fakeNode) Command(string, ...string) exec.Cmd { return nil }
func (f fakeNode) CommandContext(context.Context, string, ...string) exec.Cmd {
	return nil
}
func (f fakeNode) String() string              { return string(f) }
func (f fakeNode) Role() (string, error)       { return "", nil }
func (f fakeNode) IP() (string, string, error) { return "", "", nil }
func (f fakeNode) SerialLogs(io.Writer) error  { return nil }

func Test_selectImagesAndNodes(t *testing.T) {
	const (
		image1 = "docker.io/library/image1:tag1"
		image2 = "docker.io/library/image2:tag1"
		id1    = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
		id2    = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	)
	node1, node2 := fakeNode("node1"), fakeNode("node2")

	tests := []struct {
		name string
		// tags present per node, keyed by node name then image ID
		tags              map[string]map[string][]string
		imageNames        []string
		imageIDs          []string
		wantImagesToLoad  []string
		wantSelectedNodes []string
		wantReTagged      []string
	}{
		{
			name: "all images present on all nodes",
			tags: map[string]map[string][]string{
				"node1": {id1: {image1}, id2: {image2}},
				"node2": {id1: {image1}, id2: {image2}},
			},
			imageNames:        []string{image1, image2},
			imageIDs:          []string{id1, id2},
			wantImagesToLoad:  nil,
			wantSelectedNodes: []string{},
		},
		{
			name: "only the missing image is loaded",
			tags: map[string]map[string][]string{
				"node1": {id1: {image1}},
				"node2": {id1: {image1}},
			},
			imageNames:        []string{image1, image2},
			imageIDs:          []string{id1, id2},
			wantImagesToLoad:  []string{image2},
			wantSelectedNodes: []string{"node1", "node2"},
		},
		{
			name: "only the nodes missing an image are selected",
			tags: map[string]map[string][]string{
				"node1": {id1: {image1}, id2: {image2}},
				"node2": {id1: {image1}},
			},
			imageNames:        []string{image1, image2},
			imageIDs:          []string{id1, id2},
			wantImagesToLoad:  []string{image2},
			wantSelectedNodes: []string{"node2"},
		},
		{
			name: "present with another tag is re-tagged instead of loaded",
			tags: map[string]map[string][]string{
				"node1": {id1: {"docker.io/library/image1:other"}},
				"node2": {id1: {"docker.io/library/image1:other"}},
			},
			imageNames:        []string{image1},
			imageIDs:          []string{id1},
			wantImagesToLoad:  nil,
			wantSelectedNodes: []string{},
			wantReTagged:      []string{"node1", "node2"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tagFetcher := func(n nodes.Node, imageID string) (map[string]bool, error) {
				tags := map[string]bool{}
				for _, tag := range tt.tags[n.String()][imageID] {
					tags[tag] = true
				}
				return tags, nil
			}
			idFetcher := func(n nodes.Node, imageName string) (string, error) {
				for id, tags := range tt.tags[n.String()] {
					for _, tag := range tags {
						if tag == imageName {
							return id, nil
						}
					}
				}
				return "", errors.New("image not present")
			}
			reTagged := []string{}
			reTagger := func(n nodes.Node, _, _ string) error {
				reTagged = append(reTagged, n.String())
				return nil
			}

			imagesToLoad, selectedNodes := selectImagesAndNodes(
				log.NoopLogger{}, []nodes.Node{node1, node2},
				tt.imageNames, tt.imageIDs,
				tagFetcher, idFetcher, reTagger,
			)

			selected := []string{}
			for name := range selectedNodes {
				selected = append(selected, name)
			}
			sort.Strings(selected)
			sort.Strings(reTagged)
			if !reflect.DeepEqual(imagesToLoad, tt.wantImagesToLoad) {
				t.Errorf("imagesToLoad = %v, want %v", imagesToLoad, tt.wantImagesToLoad)
			}
			if !reflect.DeepEqual(selected, tt.wantSelectedNodes) {
				t.Errorf("selectedNodes = %v, want %v", selected, tt.wantSelectedNodes)
			}
			if len(reTagged) != len(tt.wantReTagged) {
				t.Errorf("reTagged = %v, want %v", reTagged, tt.wantReTagged)
			}
		})
	}
}

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
