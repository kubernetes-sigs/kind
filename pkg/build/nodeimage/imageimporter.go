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

package nodeimage

import (
	"io"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

type containerdImporter struct {
	containerCmder exec.Cmder
}

func newContainerdImporter(containerCmder exec.Cmder) *containerdImporter {
	return &containerdImporter{
		containerCmder: containerCmder,
	}
}

func (c *containerdImporter) Prepare() error {
	if err := c.containerCmder.Command(
		"bash", "-c", "nohup containerd > /dev/null 2>&1 &",
	).Run(); err != nil {
		return err
	}
	return nil
}

func (c *containerdImporter) WaitForReady() error {
	// ctr doesn't respect timeouts when the socket doesn't exist
	// so we'll look for the socket to exist ourselves, THEN attempt ctr info
	// TODO: we are assuming the socket path, and this is kind of hacky
	if err := c.containerCmder.Command(
		"bash", "-c", `set -e
# wait for socket to exist
for i in {0..3}; do
  if [ -S /run/containerd/containerd.sock ]; then
    break
  fi
  sleep "$i"
done
# check healthy
ctr info
`,
	).Run(); err != nil {
		return errors.Wrap(err, "failed to wait for containerd to become ready")
	}
	return nil
}

func (c *containerdImporter) End() error {
	return c.containerCmder.Command("pkill", "containerd").Run()
}

func (c *containerdImporter) Pull(image, platform string) error {
	return c.containerCmder.Command(
		"ctr", "--namespace=k8s.io", "content", "fetch", "--platform="+platform, image,
	).SetStdout(io.Discard).SetStderr(io.Discard).Run()
}

func (c *containerdImporter) LoadCommand() exec.Cmd {
	return c.containerCmder.Command(
		// TODO: ideally we do not need this in the future. we have fixed at least one image
		"ctr", "--namespace=k8s.io", "images", "import", "--label=io.cri-containerd.pinned=pinned", "--all-platforms", "--no-unpack", "--digests", "-",
	)
}

func (c *containerdImporter) Tag(src, target string) error {
	return c.containerCmder.Command(
		"ctr", "--namespace=k8s.io", "images", "tag", "--force", src, target,
	).Run()
}

func (c *containerdImporter) ListImported() ([]string, error) {
	return exec.OutputLines(c.containerCmder.Command("ctr", "--namespace=k8s.io", "images", "list", "-q"))
}
