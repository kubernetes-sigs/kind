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

package config

import (
	"fmt"
	"sigs.k8s.io/kind/pkg/util"
)

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *Config) Validate() error {
	errs := []error{}
	if c.Image == "" {
		errs = append(errs, fmt.Errorf("image is a required field"))
	}
	// TODO(bentheelder): support multiple nodes
	if c.NumNodes != 1 {
		errs = append(errs, fmt.Errorf(
			"%d nodes requested but only clusters with one node are supported currently",
			c.NumNodes,
		))
	}
	if c.NodeLifecycle != nil {
		for _, hook := range c.NodeLifecycle.PreBoot {
			if len(hook.Command) == 0 {
				errs = append(errs, fmt.Errorf(
					"preBoot hooks must set command to a non-empty value",
				))
				// we don't need to repeat this error and we don't
				// have any others for this field
				break
			}
		}
		for _, hook := range c.NodeLifecycle.PreKubeadm {
			if len(hook.Command) == 0 {
				errs = append(errs, fmt.Errorf(
					"preKubeadm hooks must set command to a non-empty value",
				))
				// we don't need to repeat this error and we don't
				// have any others for this field
				break
			}
		}
		for _, hook := range c.NodeLifecycle.PostKubeadm {
			if len(hook.Command) == 0 {
				errs = append(errs, fmt.Errorf(
					"postKubeadm hooks must set command to a non-empty value",
				))
				// we don't need to repeat this error and we don't
				// have any others for this field
				break
			}
		}
	}
	if len(errs) > 0 {
		return util.NewErrors(errs)
	}
	return nil
}
