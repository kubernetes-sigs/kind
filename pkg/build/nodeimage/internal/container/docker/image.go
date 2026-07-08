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

package docker

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// SplitImage splits an image into (registry,tag) following these cases:
//
//	alpine -> (alpine, latest)
//
//	alpine:latest -> (alpine, latest)
//
//	alpine@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913 -> (alpine, latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913)
//
//	alpine:latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913 -> (alpine, latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913)
//
// NOTE: for our purposes we consider the sha to be part of the tag, and we
// resolve the implicit :latest
func SplitImage(image string) (registry, tag string, err error) {
	// we are looking for ':' and '@'
	firstColon := strings.IndexByte(image, 58)
	firstAt := strings.IndexByte(image, 64)

	// there should be a registry before the tag, and @/: should not be the last
	// character, these cases are assumed not to exist by the rest of the code
	if firstColon == 0 || firstAt == 0 || firstColon+1 == len(image) || firstAt+1 == len(image) {
		return "", "", fmt.Errorf("unexpected image: %q", image)
	}

	// NOTE: The order of these cases matters
	// case: alpine
	if firstColon == -1 && firstAt == -1 {
		return image, "latest", nil
	}

	// case: alpine@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913
	if firstAt != -1 && firstAt < firstColon {
		return image[:firstAt], "latest" + image[firstAt:], nil
	}

	// case: alpine:latest
	// case: alpine:latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913
	return image[:firstColon], image[firstColon+1:], nil
}

// ImageInspect return low-level information on containers images
func ImageInspect(containerNameOrID, format string) ([]string, error) {
	cmd := exec.Command("docker", "image", "inspect",
		"-f", format,
		containerNameOrID, // ... against the container
	)

	return exec.OutputLines(cmd)
}

// ImageID return the Id of the container image
func ImageID(containerNameOrID string) (string, error) {
	lines, err := ImageInspect(containerNameOrID, "{{ .Id }}")
	if err != nil {
		return "", err
	}
	if len(lines) != 1 {
		return "", errors.Errorf("Docker image ID should only be one line, got %d lines", len(lines))
	}
	return lines[0], nil
}
