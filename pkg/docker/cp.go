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
	"sigs.k8s.io/kind/pkg/exec"
)

// CopyTo copies the file at hostPath to the container at destPath
func CopyTo(hostPath, containerNameOrID, destPath string) error {
	cmd := exec.Command(
		"docker", "cp",
		hostPath,                       // from the source file
		containerNameOrID+":"+destPath, // to the node, at dest
	)
	return cmd.Run()
}

// CopyFrom copies the file or dir in the container at srcPath to the host at hostPath
func CopyFrom(containerNameOrID, srcPath, hostPath string) error {
	cmd := exec.Command(
		"docker", "cp",
		containerNameOrID+":"+srcPath, // from the node, at src
		hostPath,                      // to the host
	)
	return cmd.Run()
}
