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
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// bazelBuilder implements Bits for a local Bazel build
type bazelBuilder struct {
	kubeRoot string
	arch     string
	logger   log.Logger
}

var _ Builder = &bazelBuilder{}

// NewBazelBuilder returns a new Builder backed by bazel build,
// given kubeRoot, the path to the kubernetes source directory
func NewBazelBuilder(logger log.Logger, kubeRoot, arch string) (Builder, error) {
	return &bazelBuilder{
		kubeRoot: kubeRoot,
		arch:     arch,
		logger:   logger,
	}, nil
}

// Build implements Bits.Build
func (b *bazelBuilder) Build() (Bits, error) {
	// TODO: don't cd inside the current process
	// Instead wrap subprocess calls in a shell w/ CD, or something similar
	// I think windows doesn't work regardless, so using sh should be fine
	// cd to k8s source
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	// make sure we cd back when done
	defer func() {
		// TODO(bentheelder): set return error?
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(b.kubeRoot); err != nil {
		return nil, err
	}

	version, err := sourceVersion(b.kubeRoot)
	if err != nil {
		return nil, err
	}

	// build artifacts
	cmd := exec.Command(
		"bazel", "build",
		// node installed binaries
		"//cmd/kubeadm:kubeadm", "//cmd/kubectl:kubectl", "//cmd/kubelet:kubelet",
		// and the docker images
		"//build:docker-artifacts",
	)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// https://docs.bazel.build/versions/master/output_directories.html
	binDir := filepath.Join(b.kubeRoot, "bazel-bin")
	buildDir := filepath.Join(binDir, "build")
	bazelGoosGoarch := fmt.Sprintf("linux_%s", b.arch)

	// helpers to get the binary paths which may or may not be "pure" (no cgo)
	// Except for kubelet, these are pure in Kubernetes 1.14+
	// kubelet is at bazel-bin/cmd/kubelet/kubelet since 1.14+
	// TODO: do we care about building 1.13 from source once we add support
	// for building from release binaries?
	// https://github.com/kubernetes/kubernetes/pull/73930
	strippedCommandPath := func(command string) string {
		return filepath.Join(
			binDir, "cmd", command,
			fmt.Sprintf("%s_stripped", bazelGoosGoarch), command,
		)
	}
	commandPathPureOrNot := func(command string) string {
		strippedPath := strippedCommandPath(command)
		pureStrippedPath := filepath.Join(
			binDir, "cmd", command,
			fmt.Sprintf("%s_pure_stripped", bazelGoosGoarch), command,
		)
		// if the new path doesn't exist, do the old path
		if _, err := os.Stat(pureStrippedPath); os.IsNotExist(err) {
			return strippedPath
		}
		return pureStrippedPath
	}
	//
	kubeletPath := filepath.Join(binDir, "cmd", "kubelet", "kubelet")
	oldKubeletPath := strippedCommandPath("kubelet")
	if _, err := os.Stat(kubeletPath); os.IsNotExist(err) {
		kubeletPath = oldKubeletPath
	}

	// return the paths
	return &bits{
		imagePaths: []string{
			filepath.Join(buildDir, "kube-apiserver.tar"),
			filepath.Join(buildDir, "kube-controller-manager.tar"),
			filepath.Join(buildDir, "kube-scheduler.tar"),
			filepath.Join(buildDir, "kube-proxy.tar"),
		},
		binaryPaths: []string{
			kubeletPath,
			commandPathPureOrNot("kubeadm"),
			commandPathPureOrNot("kubectl"),
		},
		version: version,
	}, nil
}
