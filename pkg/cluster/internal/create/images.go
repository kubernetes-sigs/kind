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

package create

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/container/docker"
	logutil "sigs.k8s.io/kind/pkg/log"
)

// ensureNodeImages ensures that the node images used by the create
// configuration are present
func ensureNodeImages(status *logutil.Status, cfg *config.Cluster) {
	// pull each required image
	for _, image := range requiredImages(cfg).List() {
		// prints user friendly message
		if strings.Contains(image, "@sha256:") {
			image = strings.Split(image, "@sha256:")[0]
		}
		status.Start(fmt.Sprintf("Ensuring node image (%s) 🖼", image))

		// attempt to explicitly pull the image if it doesn't exist locally
		// we don't care if this errors, we'll still try to run which also pulls
		_, _ = docker.PullIfNotPresent(image, 4)
	}
}

// requiredImages returns the set of images specified by the config
func requiredImages(cfg *config.Cluster) sets.String {
	images := sets.NewString()
	for _, node := range cfg.Nodes {
		images.Insert(node.Image)
	}
	return images
}
