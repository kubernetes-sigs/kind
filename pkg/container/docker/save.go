/*
Copyright 2018 The Kubernetes Authors.

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
	"path/filepath"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// Save saves image to dest, as in `docker save`
func Save(image, dest string) error {
	return exec.Command("docker", "save", "-o", dest, image).Run()
}

// SaveToTarball saves image into a tar archive. If successful, it will return
// the diectory and the full path to the tarball in that order.
func SaveToTarball(imageName string) (string, string, error) {
	// Create a temp directory to save the tar archive.
	dir, err := fs.TempDir("", "image-tar")
	if err != nil {
		return "", "", errors.Wrap(err, "failed to create tempdir")
	}
	imageTarPath := filepath.Join(dir, "image.tar")

	// Save container image into a tar archive.
	err = Save(imageName, imageTarPath)
	if err != nil {
		return "", "", err
	}

	return dir, imageTarPath, nil
}
