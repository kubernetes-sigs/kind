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
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// TODO(bentheelder): plumb through arch

// DockerBuildBits implements Bits for a local docker-ized make / bash build
type DockerBuildBits struct {
	kubeRoot string
	arch     string
	logger   log.Logger
}

var _ Bits = &DockerBuildBits{}

// NewDockerBuildBits returns a new Bits backed by the docker-ized build,
// given kubeRoot, the path to the kubernetes source directory
func NewDockerBuildBits(logger log.Logger, kubeRoot, arch string) (bits Bits, err error) {
	return &DockerBuildBits{
		kubeRoot: kubeRoot,
		arch:     arch,
		logger:   logger,
	}, nil
}

// Build implements Bits.Build
func (b *DockerBuildBits) Build() error {
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

	// we will pass through the environment variables, prepending defaults
	// NOTE: if env are specified multiple times the last one wins
	env := append(
		[]string{
			// ensure the build isn't especially noisy..
			"KUBE_VERBOSE=0",
			// we don't want to build these images as we don't use them ...
			"KUBE_BUILD_HYPERKUBE=n",
			"KUBE_BUILD_CONFORMANCE=n",
			// build for the host platform
			"KUBE_BUILD_PLATFORMS=" + dockerBuildOsAndArch(b.arch),
			// leverage in-tree-cloud-provider-free builds by default
			// https://github.com/kubernetes/kubernetes/pull/80353
			"GOFLAGS=-tags=providerless",
		},
		os.Environ()...,
	)
	// build binaries
	what := []string{
		// binaries we use directly
		"cmd/kubeadm",
		"cmd/kubectl",
		"cmd/kubelet",
	}
	cmd := exec.Command(
		"build/run.sh",
		"make", "all", "WHAT="+strings.Join(what, " "),
	).SetEnv(env...)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to build binaries")
	}

	// build images
	cmd = exec.Command("make", "quick-release-images").SetEnv(env...)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to build images")
	}

	// capture version info
	return buildVersionFile(b.logger, b.kubeRoot)
}

func dockerBuildOsAndArch(arch string) string {
	return "linux/" + arch
}

// Paths implements Bits.Paths
func (b *DockerBuildBits) Paths() map[string]string {
	binDir := filepath.Join(b.kubeRoot,
		"_output", "dockerized", "bin", "linux", b.arch,
	)
	return map[string]string{
		// binaries (hyperkube)
		filepath.Join(binDir, "kubeadm"): "bin/kubeadm",
		filepath.Join(binDir, "kubelet"): "bin/kubelet",
		filepath.Join(binDir, "kubectl"): "bin/kubectl",
		// version file
		filepath.Join(b.kubeRoot, "_output", "git_version"): "version",
	}
}

// ImagePaths implements Bits.ImagePaths
func (b *DockerBuildBits) ImagePaths() []string {
	imageDir := filepath.Join(b.kubeRoot,
		"_output", "release-images", b.arch,
	)
	return []string{
		filepath.Join(imageDir, "kube-apiserver.tar"),
		filepath.Join(imageDir, "kube-controller-manager.tar"),
		filepath.Join(imageDir, "kube-scheduler.tar"),
		filepath.Join(imageDir, "kube-proxy.tar"),
	}
}

// Install implements Bits.Install
func (b *DockerBuildBits) Install(install InstallContext) error {
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
