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
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"
)

// Pull pulls an image, retrying up to retries times
func Pull(logger log.Logger, image string, platform string, retries int) error {
	backoff := wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Steps:    retries,
	}
	var lastErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		logger.V(1).Infof("Trying to pull image: %s for platform %s ...", image, platform)
		if lastErr = exec.Command("docker", "pull", "--platform="+platform, image).Run(); lastErr != nil {
			logger.V(1).Infof("Failed to pull image: %q ... %v", image, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err == wait.ErrWaitTimeout {
		err = lastErr
	}
	return err
}
