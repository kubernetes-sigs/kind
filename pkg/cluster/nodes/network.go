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

package nodes

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/container/docker"
)

// CreateNetwork create a bridge network for kind's cluster
func CreateNetwork(name string) error {
	if err := docker.CreateNetwork(name); err != nil {
		return errors.Wrap(err, "error creating network")
	}

	return nil
}

// DeleteNetwork delete a bridge network
func DeleteNetwork(name string) error {
	if err := docker.DeleteNetwork(name); err != nil {
		return errors.Wrap(err, "error deleting network")
	}
	return nil
}
