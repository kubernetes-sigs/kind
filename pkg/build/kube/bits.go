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

import (
	"sync"

	"github.com/pkg/errors"
)

// Bits provides the locations of Kubernetes Binaries / Images
// needed on the cluster nodes
// Implementations should be registered with RegisterNamedBits
type Bits interface {
	// Build returns any errors encountered while building it Kubernetes.
	// Some implementations (upstream binaries) may use this step to obtain
	// an existing build isntead
	Build() error
	// Paths returns a map of path on host machine to desired path in the image
	// These paths will be populated in the image relative to some base path,
	// obtainable by NodeInstall.BasePath()
	Paths() map[string]string
	// ImagePaths returns a list of paths to image archives to be loaded into
	// the Node
	ImagePaths() []string
	// Install should install the built sources on the node, assuming paths
	// have been populated
	// TODO(bentheelder): eliminate install, make install file-copies only,
	// support cross-building
	Install(InstallContext) error
}

// InstallContext should be implemented by users of Bits
// to allow installing the bits in a Docker image
type InstallContext interface {
	// Returns the base path Paths() were populated relative to
	BasePath() string
	// Run execs (cmd, ...args) in the build container and returns error
	Run(string, ...string) error
	// CombinedOutputLines is like Run but returns the output lines
	CombinedOutputLines(string, ...string) ([]string, error)
}

// NewNamedBits returns a new Bits by named implementation
// currently this includes:
// "apt" -> NewAptBits(kubeRoot)
// "bazel" -> NewBazelBuildBits(kubeRoot)
// "docker" or "make" -> NewDockerBuildBits(kubeRoot)
func NewNamedBits(name string, kubeRoot string) (bits Bits, err error) {
	bitsImpls.Lock()
	fn, ok := bitsImpls.impls[name]
	bitsImpls.Unlock()
	if !ok {
		return nil, errors.Errorf("no Bits implementation with name: %s", name)
	}
	return fn(kubeRoot)
}

// RegisterNamedBits registers a new named Bits implementation for use from
// NewNamedBits
func RegisterNamedBits(name string, fn func(string) (Bits, error)) {
	bitsImpls.Lock()
	bitsImpls.impls[name] = fn
	bitsImpls.Unlock()
}

// NamedBitsRegistered returns true if name is in the registry backing
// NewNamedBits
func NamedBitsRegistered(name string) bool {
	var ok bool
	bitsImpls.Lock()
	_, ok = bitsImpls.impls[name]
	bitsImpls.Unlock()
	return ok
}

// internal registry of named bits implementations
var bitsImpls = struct {
	impls map[string]func(string) (Bits, error)
	sync.Mutex
}{
	impls: map[string]func(string) (Bits, error){},
}
