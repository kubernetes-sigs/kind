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

package build

import (
	"fmt"
	gobuild "go/build"
)

// KubeBits provides the locations of Kubernetes Binaries / Images
// needed on the cluster nodes and logic to install them properly
type KubeBits interface {
	// Paths returns a map of path on host to desired path in the image
	Paths() map[string]string
	// Install should install all of the bits (EG debian packages) given:
	// - the paths are populated in the image
	// - NodeInstall provides access to install on some node
	Install(NodeInstall) error
}

// NodeInstall should be implemented by users of KubeBitsProvider
// to allow installing the bits
type NodeInstall interface {
	// RunOnNode execs (cmd, ...args) on a node and returns error
	RunOnNode(string, ...string) error
}

const kubeImportPath = "k8s.io/kubernetes"

// FindKubeSource attempts to locate a kubernetes checkout using go's build package
func FindKubeSource() (root string, err error) {
	// look up the source the way go build would
	pkg, err := gobuild.Default.Import(kubeImportPath, ".", gobuild.FindOnly)
	if err == nil && maybeKubeDir(pkg.Dir) {
		return pkg.Dir, nil
	}
	return "", fmt.Errorf("could not find kubenetes source")
}

// maybeKubeDir returns true if the dir looks plausibly like a kubernetes
// source directory
func maybeKubeDir(dir string) bool {
	// TODO(bentheelder): consider adding other sanity checks
	return dir != ""
}
