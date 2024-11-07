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

package kubeconfig

import (
	"os"
	"path/filepath"
	"time"
)

const lockFileRetryAttemps = 5

// these are based on
// https://github.com/kubernetes/client-go/blob/611184f7c43ae2d520727f01d49620c7ed33412d/tools/clientcmd/loader.go#L439-L440

func lockFile(filename string) error {
	// Make sure the dir exists before we try to create a lock file.
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Retry obtaining the file lock a few times to accommodate concurrent operations.
	var lastErr error
	for i := 0; i < lockFileRetryAttemps; i++ {
		f, err := os.OpenFile(lockName(filename), os.O_CREATE|os.O_EXCL, 0)
		if err == nil {
			f.Close()
			return nil
		}

		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}

	return lastErr
}

func unlockFile(filename string) error {
	return os.Remove(lockName(filename))
}

func lockName(filename string) string {
	return filename + ".lock"
}
