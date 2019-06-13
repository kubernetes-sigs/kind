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
	"strings"

	"sigs.k8s.io/kind/pkg/container/docker"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/kind/pkg/env"
	"sigs.k8s.io/kind/pkg/exec"
)

// BazelBuildBits implements Bits for a local Bazel build
type BazelBuildBits struct {
	kubeRoot string
	// computed at build time
	paths map[string]string
}

var _ Bits = &BazelBuildBits{}

func init() {
	RegisterNamedBits("bazel", NewBazelBuildBits)
}

// NewBazelBuildBits returns a new Bits backed by bazel build,
// given kubeRoot, the path to the kubernetes source directory
func NewBazelBuildBits(kubeRoot string) (bits Bits, err error) {
	return &BazelBuildBits{
		kubeRoot: kubeRoot,
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
	os.Chdir(b.kubeRoot)
	// make sure we cd back when done
	defer os.Chdir(cwd)

	// TODO(bentheelder): we assume the host arch, but cross compiling should
	// be possible now
	arch := env.GetArch()
	bazelGoosGoarch := fmt.Sprintf("linux_%s", arch)

	// build artifacts
	cmd := exec.Command(
		"bazel", "build",
		fmt.Sprintf("--platforms=@io_bazel_rules_go//go/toolchain:%s", bazelGoosGoarch),
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
	b.paths = b.findPaths(bazelGoosGoarch)

	// capture version info
	rawVersion, err := buildVersionFile(b.kubeRoot)
	if err != nil {
		return err
	}

	// additional special handling for old kubernetes versions + bazel
	// before Kubernetes v1.12.0 kubeadm requires arch specific images, instead
	// later releases use manifest list images
	// we must re-tag them here
	ver, err := version.ParseGeneric(rawVersion)
	if err != nil {
		return err
	}
	// only < 1.12.0 has this problem
	if !ver.LessThan(version.MustParseSemantic("v1.12.0")) {
		return nil
	}

	// fix all tar files
	for path := range b.paths {
		if !strings.HasSuffix(path, ".tar") {
			continue
		}
		if err := fixOldImageTags(path, arch); err != nil {
			return err
		}
	}

	return nil
}

// fixes the missing -$arch suffix on old kubernetes image archives
func fixOldImageTags(path, arch string) error {
	// open input at path and create a fixed file at path+.fixed
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(path + ".fixed")
	if err != nil {
		return err
	}
	defer out.Close()

	// create a tarball with corrected tags
	archSuffix := "-" + arch
	repositoryFixer := func(repository string) string {
		if !strings.HasSuffix(repository, archSuffix) {
			fmt.Println("fixed: " + repository + " -> " + repository + archSuffix)
			repository = repository + archSuffix
		}
		return repository
	}
	if err := docker.EditArchiveRepositories(in, out, repositoryFixer); err != nil {
		return err
	}

	// replace the original file with the fixed file
	in.Close()
	out.Sync()
	out.Close()
	return os.Rename(out.Name(), in.Name())
}

func (b *BazelBuildBits) findPaths(bazelGoosGoarch string) map[string]string {
	// https://docs.bazel.build/versions/master/output_directories.html
	binDir := filepath.Join(b.kubeRoot, "bazel-bin")
	buildDir := filepath.Join(binDir, "build")

	// all well-known paths that have not changed
	paths := map[string]string{
		// docker images
		filepath.Join(buildDir, "kube-apiserver.tar"):          "images/kube-apiserver.tar",
		filepath.Join(buildDir, "kube-controller-manager.tar"): "images/kube-controller-manager.tar",
		filepath.Join(buildDir, "kube-scheduler.tar"):          "images/kube-scheduler.tar",
		filepath.Join(buildDir, "kube-proxy.tar"):              "images/kube-proxy.tar",
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
