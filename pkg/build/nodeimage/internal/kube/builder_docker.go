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
	"path/filepath"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/internal/version"
)

// TODO(bentheelder): plumb through arch

// dockerBuilder implements Bits for a local docker-ized make / bash build
type dockerBuilder struct {
	kubeRoot string
	arch     string
	logger   log.Logger
}

var _ Builder = &dockerBuilder{}

// NewDockerBuilder returns a new Bits backed by the docker-ized build,
// given kubeRoot, the path to the kubernetes source directory
func NewDockerBuilder(logger log.Logger, kubeRoot, arch string) (Builder, error) {
	return &dockerBuilder{
		kubeRoot: kubeRoot,
		arch:     arch,
		logger:   logger,
	}, nil
}

// Build implements Bits.Build
func (b *dockerBuilder) Build() (Bits, error) {
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

	// capture version info
	sourceVersionRaw, err := sourceVersion(b.kubeRoot)
	if err != nil {
		return nil, err
	}

	kubeVersion, err := version.ParseSemantic(sourceVersionRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse source version")
	}

	makeVars := []string{
		// ensure the build isn't especially noisy..
		"KUBE_VERBOSE=0",
		// we don't want to build these images as we don't use them ...
		"KUBE_BUILD_HYPERKUBE=n",
		"KUBE_BUILD_CONFORMANCE=n",
		// build for the host platform
		"KUBE_BUILD_PLATFORMS=" + dockerBuildOsAndArch(b.arch),
	}

	// we will pass through the environment variables, prepending defaults
	// NOTE: if env are specified multiple times the last one wins
	// NOTE: currently there are no defaults so this is essentially a deep copy
	env := append([]string{}, os.Environ()...)
	// binaries we want to build
	what := []string{
		// binaries we use directly
		"cmd/kubeadm",
		"cmd/kubectl",
		"cmd/kubelet",
	}

	// build images + binaries (binaries only on 1.21+)
	cmd := exec.Command("make",
		append(
			[]string{
				"quick-release-images",
				"KUBE_EXTRA_WHAT=" + strings.Join(what, " "),
			},
			makeVars...,
		)...,
	).SetEnv(env...)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "failed to build images")
	}

	// KUBE_EXTRA_WHAT added in this commit
	// https://github.com/kubernetes/kubernetes/commit/35061acc28a666569fdd4d1c8a7693e3c01e14be
	if kubeVersion.LessThan(version.MustParseSemantic("v1.21.0-beta.1.153+35061acc28a666")) {
		// on older versions we still need to build binaries separately
		cmd = exec.Command(
			"build/run.sh",
			append(
				[]string{
					"make",
					"all",
					"WHAT=" + strings.Join(what, " "),
				},
				makeVars...,
			)...,
		).SetEnv(env...)
		exec.InheritOutput(cmd)
		if err := cmd.Run(); err != nil {
			return nil, errors.Wrap(err, "failed to build binaries")
		}
	}

	binDir := filepath.Join(b.kubeRoot,
		"_output", "dockerized", "bin", "linux", b.arch,
	)
	imageDir := filepath.Join(b.kubeRoot,
		"_output", "release-images", b.arch,
	)

	return &bits{
		binaryPaths: []string{
			filepath.Join(binDir, "kubeadm"),
			filepath.Join(binDir, "kubelet"),
			filepath.Join(binDir, "kubectl"),
		},
		imagePaths: []string{
			filepath.Join(imageDir, "kube-apiserver.tar"),
			filepath.Join(imageDir, "kube-controller-manager.tar"),
			filepath.Join(imageDir, "kube-scheduler.tar"),
			filepath.Join(imageDir, "kube-proxy.tar"),
		},
		version: sourceVersionRaw,
	}, nil
}

func dockerBuildOsAndArch(arch string) string {
	return "linux/" + arch
}
