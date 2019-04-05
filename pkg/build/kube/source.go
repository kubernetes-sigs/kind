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
	"go/build"

	"github.com/pkg/errors"
)

// ImportPath is the canonical import path for the kubernetes root package
// this is used by FindSource
const ImportPath = "k8s.io/kubernetes"

// FindSource attempts to locate a kubernetes checkout using go's build package
func FindSource() (root string, err error) {
	// look up the source the way go build would
	pkg, err := build.Default.Import(ImportPath, build.Default.GOPATH, build.FindOnly)
	if err == nil && maybeKubeDir(pkg.Dir) {
		return pkg.Dir, nil
	}
	return "", errors.New("could not find kubernetes source")
}

// maybeKubeDir returns true if the dir looks plausibly like a kubernetes
// source directory
func maybeKubeDir(dir string) bool {
	// TODO(bentheelder): consider adding other sanity checks
	return dir != ""
}
