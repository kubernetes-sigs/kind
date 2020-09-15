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

package nodeutils

import (
	"bytes"
	"encoding/json"
	"io"
	"path"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// KubeVersion returns the Kubernetes version installed on the node
func KubeVersion(n nodes.Node) (version string, err error) {
	// grab kubernetes version from the node image
	cmd := n.Command("cat", "/kind/version")
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get file")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("file should only be one line, got %d lines", len(lines))
	}
	return lines[0], nil
}

// WriteFile writes content to dest on the node
func WriteFile(n nodes.Node, dest, content string) error {
	// create destination directory
	err := n.Command("mkdir", "-p", path.Dir(dest)).Run()
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", path.Dir(dest))
	}

	return n.Command("cp", "/dev/stdin", dest).SetStdin(strings.NewReader(content)).Run()
}

// CopyNodeToNode copies file from a to b
func CopyNodeToNode(a, b nodes.Node, file string) error {
	// create destination directory
	err := b.Command("mkdir", "-p", path.Dir(file)).Run()
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %q", path.Dir(file))
	}

	// TODO: experiment with streaming instead to avoid the copy
	// for now we only use this for small files so it's not worth the complexity
	var buff bytes.Buffer
	if err := a.Command("cat", file).SetStdout(&buff).Run(); err != nil {
		return errors.Wrapf(err, "failed to read %q from node", file)
	}
	if err := b.Command("cp", "/dev/stdin", file).SetStdin(&buff).Run(); err != nil {
		return errors.Wrapf(err, "failed to write %q to node", file)
	}

	return nil
}

// LoadImageArchive loads image onto the node, where image is a Reader over an image archive
func LoadImageArchive(n nodes.Node, image io.Reader) error {
	cmd := n.Command("ctr", "--namespace=k8s.io", "images", "import", "-").SetStdin(image)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to load image")
	}
	return nil
}

// ImageID returns ID of image on the node with the given image name if present
func ImageID(n nodes.Node, image string) (string, error) {
	var out bytes.Buffer
	if err := n.Command("crictl", "inspecti", image).SetStdout(&out).Run(); err != nil {
		return "", err
	}
	// we only care about the image ID
	crictlOut := struct {
		Status struct {
			ID string `json:"id"`
		} `json:"status"`
	}{}
	if err := json.Unmarshal(out.Bytes(), &crictlOut); err != nil {
		return "", err
	}
	return crictlOut.Status.ID, nil
}
