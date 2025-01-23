/*
Copyright 2019 The Kubernetes Authors.

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

import (
	"io"

	"sigs.k8s.io/kind/pkg/exec"
)

type ContainerdImporter struct {
	containerCmder exec.Cmder
}

func NewContainerdImporter(containerCmder exec.Cmder) *ContainerdImporter {
	return &ContainerdImporter{
		containerCmder: containerCmder,
	}
}

func (c *ContainerdImporter) Prepare() error {
	if err := c.containerCmder.Command(
		"bash", "-c", "nohup containerd > /dev/null 2>&1 &",
	).Run(); err != nil {
		return err
	}
	// TODO(bentheelder): some healthcheck?
	return nil
}

func (c *ContainerdImporter) End() error {
	return c.containerCmder.Command("pkill", "containerd").Run()
}

func (c *ContainerdImporter) Pull(image, platform string) error {
	return c.containerCmder.Command(
		"ctr", "--namespace=k8s.io", "content", "fetch", "--platform="+platform, image,
	).SetStdout(io.Discard).SetStderr(io.Discard).Run()
}

func (c *ContainerdImporter) LoadCommand() exec.Cmd {
	return c.containerCmder.Command(
		// TODO: ideally we do not need this in the future. we have fixed at least one image
		"ctr", "--namespace=k8s.io", "images", "import", "--label=io.cri-containerd.pinned=pinned", "--all-platforms", "--no-unpack", "--digests", "-",
	)
}

func (c *ContainerdImporter) Tag(src, target string) error {
	return c.containerCmder.Command(
		"ctr", "--namespace=k8s.io", "images", "tag", "--force", src, target,
	).Run()
}

func (c *ContainerdImporter) ListImported() ([]string, error) {
	return exec.OutputLines(c.containerCmder.Command("ctr", "--namespace=k8s.io", "images", "list", "-q"))
}
