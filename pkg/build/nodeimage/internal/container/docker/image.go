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
//	localhost:5000/foo:bar -> (localhost:5000/foo, bar)
//
// NOTE: for our purposes we consider the sha to be part of the tag, and we
// resolve the implicit :latest. A colon before the last slash is a registry
// port, not a tag.
func SplitImage(image string) (registry, tag string, err error) {
	// A leading or trailing separator is not a reference. An empty name is not
	// either. The rest of the builder assumes these do not occur.
	if image == "" ||
		image[0] == ':' || image[0] == '@' ||
		image[len(image)-1] == ':' || image[len(image)-1] == '@' {
		return "", "", fmt.Errorf("unexpected image: %q", image)
	}

	// A digest colon (sha256:...) is not a tag. Split it off before looking
	// for the tag so a registry port is not confused with either one.
	name := image
	digest := ""
	if at := strings.IndexByte(image, '@'); at != -1 {
		name = image[:at]
		digest = image[at:]
		if name == "" || len(digest) < 2 {
			return "", "", fmt.Errorf("unexpected image: %q", image)
		}
	}

	// A colon before the last slash is a registry port (localhost:5000/foo).
	// The tag colon, when present, is the last colon after that slash.
	lastSlash := strings.LastIndexByte(name, '/')
	lastColon := strings.LastIndexByte(name, ':')
	if lastColon > lastSlash {
		registry = name[:lastColon]
		tag = name[lastColon+1:]
		if registry == "" || tag == "" {
			return "", "", fmt.Errorf("unexpected image: %q", image)
		}
	} else {
		registry = name
		tag = "latest"
	}
	if digest != "" {
		tag += digest
	}
	return registry, tag, nil
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
