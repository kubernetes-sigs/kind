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

// Package docker contains helpers for working with docker
package docker

import (
	"sigs.k8s.io/kind/pkg/exec"
)

// PullIfNotPresent will pull an image if it is not present locally
// it returns true if it attempted to pull, and any errors from pulling
func PullIfNotPresent(image string) (pulled bool, err error) {
	// if this did not return an error, then the image exists locally
	cmd := exec.Command("docker", "inspect", "--type=image", image)
	if err := cmd.Run(); err == nil {
		return false, nil
	}
	// otherwise try to pull it
	return true, exec.Command("docker", "pull", image).Run()
}
