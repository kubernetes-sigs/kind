/*
Copyright 2020 The Kubernetes Authors.

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

package nodeimage

import (
	"sigs.k8s.io/kind/pkg/build/nodeimage/internal/kube"
	"sigs.k8s.io/kind/pkg/exec"
)

// private kube.InstallContext implementation, local to the image build
type installContext struct {
	basePath    string
	containerID string
}

var _ kube.InstallContext = &installContext{}

func (ic *installContext) BasePath() string {
	return ic.basePath
}

func (ic *installContext) Run(command string, args ...string) error {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", ic.containerID, command},
			args...,
		)...,
	)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (ic *installContext) CombinedOutputLines(command string, args ...string) ([]string, error) {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", ic.containerID, command},
			args...,
		)...,
	)
	return exec.CombinedOutputLines(cmd)
}
