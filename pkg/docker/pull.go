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

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/exec"
)

// PullIfNotPresent will pull an image if it is not present locally
// retrying up to retries times
// it returns true if it attempted to pull, and any errors from pulling
func PullIfNotPresent(image string, retries int) (pulled bool, err error) {
	// TODO(bentheelder): switch most (all) of the logging here to debug level
	// once we have configurable log levels
	// if this did not return an error, then the image exists locally
	cmd := exec.Command("docker", "inspect", "--type=image", image)
	if err := cmd.Run(); err == nil {
		log.Infof("Image: %s present locally", image)
		return false, nil
	}
	// otherwise try to pull it
	return true, Pull(image, retries)
}

// Pull pulls an image, retrying up to retries times
func Pull(image string, retries int) error {
	log.Infof("Pulling image: %s ...", image)
	err := exec.Command("docker", "pull", image).Run()
	// retry pulling up to retries times if necessary
	if err != nil {
		for i := 0; i < retries; i++ {
			time.Sleep(time.Second * time.Duration(i+1))
			log.WithError(err).Infof("Trying again to pull image: %s ...", image)
			// TODO(bentheelder): add some backoff / sleep?
			err = exec.Command("docker", "pull", image).Run()
			if err == nil {
				break
			}
		}
	}
	if err != nil {
		log.WithError(err).Infof("Failed to pull image: %s", image)
	}
	return err
}
