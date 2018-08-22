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

import "path/filepath"

// BazelBuildBits implements KubeBits for a local Bazel build
type BazelBuildBits struct {
	paths map[string]string
}

var _ KubeBits = &BazelBuildBits{}

func (l *BazelBuildBits) Paths() map[string]string {
	// TODO(bentheelder): maybe copy the map before returning /shrug
	return l.paths
}

func NewBazelBuildBits(kubeRoot string) (bits KubeBits, err error) {
	// https://docs.bazel.build/versions/master/output_directories.html
	binDir := filepath.Join(kubeRoot, "bazel-bin")
	bits = &BazelBuildBits{
		paths: map[string]string{
			// debians
			filepath.Join(binDir, "build", "debs", "kubeadm.deb"):        "debs/kubeadm.deb",
			filepath.Join(binDir, "build", "debs", "kubelet.deb"):        "debs/kubelet.deb",
			filepath.Join(binDir, "build", "debs", "kubectl.deb"):        "debs/kubectl.deb",
			filepath.Join(binDir, "build", "debs", "kubernetes-cni.deb"): "debs/kubernetes-cni.deb",
			filepath.Join(binDir, "build", "debs", "cri-tools.deb"):      "debs/cri-tools.deb",
			// docker images
			filepath.Join(binDir, "build", "kube-proxy.tar"):              "images/kube-proxy.tar",
			filepath.Join(binDir, "build", "kube-controller-manager.tar"): "images/kube-controller-manager.tar",
			filepath.Join(binDir, "build", "kube-scheduler.tar"):          "images/kube-scheduler.tar",
			filepath.Join(binDir, "build", "kube-apiserver.tar"):          "images/kube-apiserver.tar",
		},
	}
	return bits, nil
}
