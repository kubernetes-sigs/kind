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
	"path"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// BazelBuildBits implements Bits for a local Bazel build
type BazelBuildBits struct {
	kubeRoot string
	arch     string
	logger   log.Logger
	// computed at build time
	paths      map[string]string
	imagePaths []string
}

var _ Bits = &BazelBuildBits{}

// NewBazelBuildBits returns a new Bits backed by bazel build,
// given kubeRoot, the path to the kubernetes source directory
func NewBazelBuildBits(logger log.Logger, kubeRoot, arch string) (bits Bits, err error) {
	return &BazelBuildBits{
		kubeRoot: kubeRoot,
		arch:     arch,
		logger:   logger,
	}, nil
}

// Build implements Bits.Build
func (b *BazelBuildBits) Build() error {
	// TODO(bentheelder): support other modes of building
	// cd to k8s source
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	// make sure we cd back when done
	defer func() {
		// TODO(bentheelder): set return error?
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(b.kubeRoot); err != nil {
		return err
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
		return err
	}

	// capture the output paths
	bazelGoosGoarch := fmt.Sprintf("linux_%s", b.arch)
	b.paths = b.findPaths(bazelGoosGoarch)

	// capture version info
	return buildVersionFile(b.logger, b.kubeRoot)
}

func (b *BazelBuildBits) findPaths(bazelGoosGoarch string) map[string]string {
	// https://docs.bazel.build/versions/master/output_directories.html
	binDir := filepath.Join(b.kubeRoot, "bazel-bin")
	buildDir := filepath.Join(binDir, "build")

	// docker images
	b.imagePaths = []string{
		filepath.Join(buildDir, "kube-apiserver.tar"),
		filepath.Join(buildDir, "kube-controller-manager.tar"),
		filepath.Join(buildDir, "kube-scheduler.tar"),
		filepath.Join(buildDir, "kube-proxy.tar"),
	}

	// all well-known paths that have not changed
	paths := map[string]string{
		// version file
		filepath.Join(b.kubeRoot, "_output", "git_version"): "version",
	}

	// binaries that may be in different locations
	kubeadmPureStrippedPath := filepath.Join(
		binDir, "cmd", "kubeadm",
		fmt.Sprintf("%s_pure_stripped", bazelGoosGoarch), "kubeadm",
	)
	kubeadmStrippedPath := filepath.Join(
		binDir, "cmd", "kubeadm",
		fmt.Sprintf("%s_stripped", bazelGoosGoarch), "kubeadm",
	)
	kubectlPureStrippedPath := filepath.Join(
		binDir, "cmd", "kubectl",
		fmt.Sprintf("%s_pure_stripped", bazelGoosGoarch), "kubectl",
	)
	kubectlStrippedPath := filepath.Join(
		binDir, "cmd", "kubectl",
		fmt.Sprintf("%s_stripped", bazelGoosGoarch), "kubectl",
	)
	oldKubeletPath := filepath.Join(
		binDir, "cmd", "kubelet",
		fmt.Sprintf("%s_stripped", bazelGoosGoarch), "kubelet",
	)
	newKubeletPath := filepath.Join(binDir, "cmd", "kubelet", "kubelet")

	// look for one path then fall back to the alternate for each
	if _, err := os.Stat(kubeadmPureStrippedPath); os.IsNotExist(err) {
		paths[kubeadmStrippedPath] = "bin/kubeadm"
	} else {
		paths[kubeadmPureStrippedPath] = "bin/kubeadm"
	}
	if _, err := os.Stat(kubectlPureStrippedPath); os.IsNotExist(err) {
		paths[kubectlStrippedPath] = "bin/kubectl"
	} else {
		paths[kubectlPureStrippedPath] = "bin/kubectl"
	}
	if _, err := os.Stat(oldKubeletPath); os.IsNotExist(err) {
		paths[newKubeletPath] = "bin/kubelet"
	} else {
		paths[oldKubeletPath] = "bin/kubelet"
	}

	return paths
}

// Paths implements Bits.Paths
func (b *BazelBuildBits) Paths() map[string]string {
	return b.paths
}

// ImagePaths implements Bits.ImagePaths
func (b *BazelBuildBits) ImagePaths() []string {
	return b.imagePaths
}

// Install implements Bits.Install
func (b *BazelBuildBits) Install(install InstallContext) error {
	kindBinDir := path.Join(install.BasePath(), "bin")

	// symlink the kubernetes binaries into $PATH
	binaries := []string{"kubeadm", "kubelet", "kubectl"}
	for _, binary := range binaries {
		if err := install.Run("ln", "-s",
			path.Join(kindBinDir, binary),
			path.Join("/usr/bin/", binary),
		); err != nil {
			return errors.Wrap(err, "failed to symlink binaries")
		}
	}

	return nil
}
