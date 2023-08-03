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
	"path"
	"regexp"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// createFile creates the file at filePath in the container,
// ensuring the directory exists and writing contents to the file
func createFile(containerCmder exec.Cmder, filePath, contents string) error {
	// NOTE: the paths inside the container should use the path package
	// and not filepath (!), we want posixy paths in the linux container, NOT
	// whatever path format the host uses. For paths on the host we use filepath
	if err := containerCmder.Command("mkdir", "-p", path.Dir(filePath)).Run(); err != nil {
		return err
	}

	return containerCmder.Command(
		"cp", "/dev/stdin", filePath,
	).SetStdin(
		strings.NewReader(contents),
	).Run()
}

func findSandboxImage(config string) (string, error) {
	match := regexp.MustCompile(`sandbox_image\s+=\s+"([^\n]+)"`).FindStringSubmatch(config)
	if len(match) < 2 {
		return "", errors.New("failed to parse sandbox_image from config")
	}
	return match[1], nil
}

func dockerBuildOsAndArch(arch string) string {
	return "linux/" + arch
}
