/*
Copyright The Kubernetes Authors.

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
	"bufio"
	"bytes"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

// QueryContentBlobs checks which blobs exist on node
func QueryContentBlobs(n nodes.Node, digests []string) (map[string]bool, error) {
	allBlobs, err := ListContentBlobs(n)
	if err != nil {
		return nil, err
	}

	blobSet := make(map[string]bool, len(allBlobs))
	for _, blob := range allBlobs {
		blobSet[blob] = true
	}

	result := make(map[string]bool, len(digests))
	for _, digest := range digests {
		result[digest] = blobSet[digest]
	}

	return result, nil
}

// ListContentBlobs lists all blobs in content store
func ListContentBlobs(n nodes.Node) ([]string, error) {
	var out bytes.Buffer
	cmd := n.Command("ctr", "--namespace=k8s.io", "content", "ls", "--quiet")
	cmd.SetStdout(&out)

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "failed to list content store")
	}

	digests := []string{}
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "sha256:") {
			digests = append(digests, line)
		}
	}

	return digests, scanner.Err()
}
