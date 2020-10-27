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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/errors"
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

	kubeletPath, err := findGoBinary("//cmd/kubelet:kubelet")
	if err != nil {
		// This rule was previously aliased. To handle older versions of k/k, we
		// manual dereference of the alias created here:
		//
		// https://github.com/kubernetes/kubernetes/blob/b56d0acaf5bead31bd17d3b88d4a167fcbac7866/build/go.bzl#L45
		kubeletPath, err = findGoBinary("//cmd/kubelet:_kubelet-cgo")
		if err != nil {
			return nil, errors.Wrap(err, "could not find kubelet")
		}
	}

	kubeadmPath, err := findGoBinary("//cmd/kubeadm:kubeadm")
	if err != nil {
		return nil, errors.Wrap(err, "could not find kubeadm")
	}

	kubectlPath, err := findGoBinary("//cmd/kubectl:kubectl")
	if err != nil {
		return nil, errors.Wrap(err, "could not find kubectl")
	}

	// https://docs.bazel.build/versions/master/output_directories.html
	buildDir := filepath.Join(b.kubeRoot, "bazel-bin/build")

	// return the paths
	return &bits{
		imagePaths: []string{
			filepath.Join(buildDir, "kube-apiserver.tar"),
			filepath.Join(buildDir, "kube-controller-manager.tar"),
			filepath.Join(buildDir, "kube-scheduler.tar"),
			filepath.Join(buildDir, "kube-proxy.tar"),
		},
		binaryPaths: []string{
			filepath.Join(b.kubeRoot, kubeletPath),
			filepath.Join(b.kubeRoot, kubeadmPath),
			filepath.Join(b.kubeRoot, kubectlPath),
		},
		version: version,
	}, nil
}

func findGoBinary(label string) (string, error) {
	// This output of bazel aquery --output=jsonproto is an ActionGraphContainer
	// as defined in:
	//
	// https://cs.opensource.google/bazel/bazel/+/master:src/main/protobuf/analysis.proto
	type (
		Action struct {
			Mnemonic  string   `json:"mnemonic"`
			OutputIDs []string `json:"outputIds"`
		}
		Artifact struct {
			ID       string `json:"id"`
			ExecPath string `json:"execPath"`
		}
		ActionGraphContainer struct {
			Artifacts []Artifact `json:"artifacts"`
			Actions   []Action   `json:"actions"`
		}
	)

	cmd := exec.Command("bazel", "aquery", "--output=jsonproto", label)
	exec.InheritOutput(cmd)

	actionBytes, err := exec.Output(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to query action graph")
	}
	var agc ActionGraphContainer
	if err := json.Unmarshal(actionBytes, &agc); err != nil {
		return "", errors.Wrap(err, "failed to unpack action graph container")
	}

	var linkActions []Action
	for _, action := range agc.Actions {
		if action.Mnemonic == "GoLink" {
			linkActions = append(linkActions, action)
		}
	}
	if len(linkActions) != 1 {
		return "", fmt.Errorf("unexpected number of link actions %d, wanted 1", len(linkActions))
	}
	linkAction := linkActions[0]
	if len(linkAction.OutputIDs) != 1 {
		return "", fmt.Errorf("unexpected number of link action outputs %d, wanted 1", len(linkAction.OutputIDs))
	}
	outputID := linkAction.OutputIDs[0]

	for _, artifact := range agc.Artifacts {
		if artifact.ID == outputID {
			return artifact.ExecPath, nil
		}
	}
	// We really should never get here
	return "", fmt.Errorf("could not find artifact corresponding to output id %q", outputID)
}
