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
	"fmt"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/util/version"

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

func repositoryCorrectorForVersion(kubeVersion *version.Version, arch string) func(string) string {
	archSuffix := "-" + arch

	// For kubernetes v1.15+ (actually 1.16 alpha versions) we may need to
	// drop the arch suffix from images to get the expected image
	// for < v1.12 we need to do the opposite.
	// We can accomplish this by just handling < 1.12 & >= 1.12 as we won't
	// touch images that match the expectation in either case ...
	if kubeVersion.LessThan(version.MustParseSemantic("v1.12.0")) {
		return func(repository string) string {
			if !strings.HasSuffix(repository, archSuffix) {
				fixed := repository + archSuffix
				fmt.Println("fixed: " + repository + " -> " + fixed)
				repository = fixed
			}
			return repository
		}
	}

	return func(repository string) string {
		if strings.HasSuffix(repository, archSuffix) {
			fixed := strings.TrimSuffix(repository, archSuffix)
			fmt.Println("fixed: " + repository + " -> " + fixed)
			repository = fixed
		}
		return repository
	}
}

func findSandboxImage(images []string) string {
	for _, image := range images {
		// yep this seems legit
		// https://github.com/kubernetes-sigs/kind/issues/1471#issuecomment-617579803
		if strings.Contains(image, "pause") {
			return image
		}
	}
	return ""
}
