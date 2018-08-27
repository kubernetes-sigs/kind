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

package build

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"k8s.io/test-infra/kind/pkg/exec"
)

// TempDir is like ioutil.TempDir, but more docker friendly
func TempDir(dir, prefix string) (name string, err error) {
	name, err = ioutil.TempDir(dir, prefix)
	if err != nil {
		return "", err
	}
	// on macOS $TMPDIR is typically /var/..., which is not mountable
	// /private/var/... is the mountable equivalent
	if runtime.GOOS == "darwin" && strings.HasPrefix(name, "/var/") {
		name = filepath.Join("/private", name)
	}
	return name, nil
}

// TODO(bentheelder): vendor a portable go library for this and use instead
func copyDir(src, dst string) error {
	src = filepath.Clean(src) + string(filepath.Separator) + "."
	dst = filepath.Clean(dst)
	cmd := exec.Command("cp", "-r", src, dst)
	cmd.Debug = true
	cmd.InheritOutput = true
	return cmd.Run()
}

// TODO(bentheelder): vendor a portable go library for this and use instead
func copyFile(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
		return err
	}
	cmd := exec.Command("cp", src, dst)
	cmd.Debug = true
	cmd.InheritOutput = true
	return cmd.Run()
}
