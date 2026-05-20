/*
Copyright 2026 The Kubernetes Authors.

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

package swarm

import (
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cli"
)

// ensureNodeImages ensures that the node images used by the create
// configuration are present on every swarm host.
func ensureNodeImages(logger log.Logger, status *cli.Status, cfg *config.Cluster, hosts []Host) error {
	for _, image := range common.RequiredNodeImages(cfg).List() {
		friendlyImageName, image := sanitizeImage(image)
		status.Start(fmt.Sprintf("Ensuring node image (%s) on %d host(s) 🖼", friendlyImageName, len(hosts)))
		for _, h := range hosts {
			if _, err := pullIfNotPresent(logger, h.Context, image, 4); err != nil {
				status.End(false)
				return err
			}
		}
	}
	return nil
}

// pullIfNotPresent will pull an image on the given docker context if it
// is not present locally there, retrying up to retries times.
func pullIfNotPresent(logger log.Logger, ctxName, image string, retries int) (pulled bool, err error) {
	cmd := exec.Command("docker",
		dockerArgs(ctxName, "inspect", "--type=image", image)...,
	)
	if err := cmd.Run(); err == nil {
		logger.V(1).Infof("Image: %s present on %s", image, ctxName)
		return false, nil
	}
	return true, pull(logger, ctxName, image, retries)
}

func pull(logger log.Logger, ctxName, image string, retries int) error {
	logger.V(1).Infof("Pulling image %s on %s ...", image, ctxName)
	err := exec.Command("docker", dockerArgs(ctxName, "pull", image)...).Run()
	for i := 0; err != nil && i < retries; i++ {
		time.Sleep(time.Second * time.Duration(i+1))
		logger.V(1).Infof("Trying again to pull image %q on %s ... %v", image, ctxName, err)
		err = exec.Command("docker", dockerArgs(ctxName, "pull", image)...).Run()
	}
	return errors.Wrapf(err, "failed to pull image %q on %s", image, ctxName)
}

func sanitizeImage(image string) (string, string) {
	if strings.Contains(image, "@sha256:") {
		return strings.Split(image, "@sha256:")[0], image
	}
	return image, image
}
