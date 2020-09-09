/*
Copyright 2018 The Kubernetes Authors.

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

package kube

// Bits provides the locations of Kubernetes Binaries / Images
// needed on the cluster nodes
// Implementations should be registered with RegisterNamedBits
type Bits interface {
	// BinaryPaths returns a list of paths to binaries on the host machine that
	// should be added to PATH in the Node image
	BinaryPaths() []string
	// ImagePaths returns a list of paths to image archives to be loaded into
	// the Node
	ImagePaths() []string
	// Version
	Version() string
}

// shared real bits implementation for now

type bits struct {
	// computed at build time
	binaryPaths []string
	imagePaths  []string
	version     string
}

var _ Bits = &bits{}

func (b *bits) BinaryPaths() []string {
	return b.binaryPaths
}

func (b *bits) ImagePaths() []string {
	return b.imagePaths
}

func (b *bits) Version() string {
	return b.version
}
