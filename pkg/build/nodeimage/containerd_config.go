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
	"bytes"
	"text/template"

	"sigs.k8s.io/kind/pkg/errors"
)

type containerdConfigTemplateData struct {
	SandboxImage string
}

const containerdConfigTemplate = `# explicitly use v2 config format
version = 2

# set default runtime handler to v2, which has a per-pod shim
[plugins."io.containerd.grpc.v1.cri".containerd]
  default_runtime_name = "runc"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"

# Setup a runtime with the magic name ("test-handler") used for Kubernetes
# runtime class tests ...
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.test-handler]
  runtime_type = "io.containerd.runc.v2"

[plugins."io.containerd.grpc.v1.cri"]
  # ensure the sandbox image matches kubeadm
  # TODO: probably we should instead just use the containerd default
  # Implementing the pod sandbox is a CRI implementation detail ...
  sandbox_image = "{{.SandboxImage}}"
  # allow hugepages controller to be missing
  # see https://github.com/containerd/cri/pull/1501
  tolerate_missing_hugepages_controller = true
`

func getContainerdConfig(data containerdConfigTemplateData) (string, error) {
	t, err := template.New("containerd-config").Parse(containerdConfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse containerd config template")
	}
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}
	return buff.String(), nil
}
