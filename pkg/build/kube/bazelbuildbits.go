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
	"os"
	"path"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"k8s.io/test-infra/kind/pkg/exec"
)

// BazelBuildBits implements Bits for a local Bazel build
type BazelBuildBits struct {
	kubeRoot string
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

	// build artifacts
	cmd := exec.Command("bazel", "build")
	cmd.Args = append(cmd.Args,
		// TODO(bentheelder): we assume linux amd64, but we could select
		// this based on Arch etc. throughout, this flag supports GOOS/GOARCH
		"--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64",
		// we want the debian packages
		"//build/debs:debs",
		// and the docker images
		//"//cluster/images/hyperkube:hyperkube.tar",
		"//build:docker-artifacts",
	)
	cmd.Debug = true
	cmd.InheritOutput = true
	if err := cmd.Run(); err != nil {
		return err
	}

	// capture version info
	return buildVersionFile(b.kubeRoot)
}

// Paths implements Bits.Paths
func (b *BazelBuildBits) Paths() map[string]string {
	// https://docs.bazel.build/versions/master/output_directories.html
	binDir := filepath.Join(b.kubeRoot, "bazel-bin")
	buildDir := filepath.Join(binDir, "build")
	return map[string]string{
		// debians
		filepath.Join(buildDir, "debs", "kubeadm.deb"):        "debs/kubeadm.deb",
		filepath.Join(buildDir, "debs", "kubelet.deb"):        "debs/kubelet.deb",
		filepath.Join(buildDir, "debs", "kubectl.deb"):        "debs/kubectl.deb",
		filepath.Join(buildDir, "debs", "kubernetes-cni.deb"): "debs/kubernetes-cni.deb",
		filepath.Join(buildDir, "debs", "cri-tools.deb"):      "debs/cri-tools.deb",
		// docker images
		filepath.Join(buildDir, "kube-apiserver.tar"):          "images/kube-apiserver.tar",
		filepath.Join(buildDir, "kube-controller-manager.tar"): "images/kube-controller-manager.tar",
		filepath.Join(buildDir, "kube-scheduler.tar"):          "images/kube-scheduler.tar",
		filepath.Join(buildDir, "kube-proxy.tar"):              "images/kube-proxy.tar",
		// version file
		filepath.Join(b.kubeRoot, "_output", "git_version"): "version",
	}
}

// Install implements Bits.Install
func (b *BazelBuildBits) Install(install InstallContext) error {
	base := install.BasePath()

	debs := path.Join(base, "debs", "*.deb")

	if err := install.Run("/bin/sh", "-c", "dpkg -i "+debs); err != nil {
		log.Errorf("Image install failed! %v", err)
		return err
	}

	if err := install.Run("/bin/sh", "-c",
		"rm -rf /kind/bits/debs/*.deb"+
			" /var/cache/debconf/* /var/lib/apt/lists/* /var/log/*kg",
	); err != nil {
		log.Errorf("Image install failed! %v", err)
		return err
	}

	return nil
}
