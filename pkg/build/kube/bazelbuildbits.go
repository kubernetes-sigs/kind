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

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/util"
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
	bazelGoosGoarch := fmt.Sprintf("linux_%s", util.GetArch())

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
	return buildVersionFile(b.kubeRoot)
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
		// borrow kubelet service files from bazel debians
		// TODO(bentheelder): probably we should use our own config instead :-)
		filepath.Join(b.kubeRoot, "build", "debs", "kubelet.service"): "systemd/kubelet.service",
		filepath.Join(b.kubeRoot, "build", "debs", "10-kubeadm.conf"): "systemd/10-kubeadm.conf",
		// binaries
		filepath.Join(
			binDir, "cmd", "kubeadm",
			// pure-go binary
			fmt.Sprintf("%s_pure_stripped", bazelGoosGoarch), "kubeadm",
		): "bin/kubeadm",
		filepath.Join(
			binDir, "cmd", "kubectl",
			// pure-go binary
			fmt.Sprintf("%s_pure_stripped", bazelGoosGoarch), "kubectl",
		): "bin/kubectl",
	}

	// paths that changed: kubelet binary
	oldKubeletPath := filepath.Join(
		binDir, "cmd", "kubelet",
		// cgo binary
		fmt.Sprintf("%s_stripped", bazelGoosGoarch), "kubelet",
	)
	newKubeletPath := filepath.Join(binDir, "cmd", "kubelet", "kubelet")
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

	// enable the kubelet service
	kubeletService := path.Join(install.BasePath(), "systemd/kubelet.service")
	if err := install.Run("systemctl", "enable", kubeletService); err != nil {
		return errors.Wrap(err, "failed to enable kubelet service")
	}

	// setup the kubelet dropin
	kubeletDropinSource := path.Join(install.BasePath(), "systemd/10-kubeadm.conf")
	kubeletDropin := "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"
	if err := install.Run("mkdir", "-p", path.Dir(kubeletDropin)); err != nil {
		return errors.Wrap(err, "failed to configure kubelet service")
	}
	if err := install.Run("cp", kubeletDropinSource, kubeletDropin); err != nil {
		return errors.Wrap(err, "failed to configure kubelet service")
	}
	return nil
}
