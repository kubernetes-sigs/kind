/*
Copyright 2023 The Kubernetes Authors.

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

// Package cloudprovider implements the external cloud provider configuration action
package cloudprovider

import (
	"fmt"
	"os/exec"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
)

const (
	clusterLabelKey = "io.x-k8s.kind.cluster"
	image           = "kindest/cloud-provider:test"
)

// Action implements and action for configuring and starting the
// external load balancer in front of the control-plane nodes.
type Action struct{}

// NewAction returns a new Action for configuring the load balancer
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	networkName := "kind"
	clusterName := ctx.Config.Name
	args := []string{
		"--detach",
		"--restart=on-failure:1",
		"--privileged",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"--label", fmt.Sprintf("%s=%s", clusterLabelKey, clusterName),
		// user a user defined docker network so we get embedded DNS
		"--net", networkName,
	}

	args = append(args, image)
	args = append(args, []string{"-cluster", clusterName}...)
	if err := exec.Command("docker", append([]string{"run", "--name", clusterName + "-cloud-provider"}, args...)...).Run(); err != nil {
		return err
	}
	ctx.Status.End(true)
	return nil
}
