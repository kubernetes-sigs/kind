/*
Copyright 2022 The Kubernetes Authors.

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
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/internal/patch"
)

const containerdConfigPath = "/etc/containerd/config.toml"

const containerdConfigPatchSystemdCgroupFalse = `
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
  SystemdCgroup = false

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test-handler.options]
  SystemdCgroup = false
`

func configureContainerdSystemdCgroupFalse(containerCmdr exec.Cmder, config string) error {
	patched, err := patch.TOML(config, []string{containerdConfigPatchSystemdCgroupFalse}, []string{})
	if err != nil {
		return errors.Wrap(err, "failed to configure containerd SystemdCgroup=false")
	}
	err = containerCmdr.Command(
		"cp", "/dev/stdin", containerdConfigPath,
	).SetStdin(strings.NewReader(patched)).Run()
	if err != nil {
		return errors.Wrap(err, "failed to configure containerd SystemdCgroup=false")
	}
	return nil
}
