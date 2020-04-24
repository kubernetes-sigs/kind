/*
Copyright 2020 The Kubernetes Authors.

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
	"regexp"

	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// TODO: we'll probably allow configuring this
//
// however currently picking a single network is equivalent to the previous
// behavior *except* that we moved from the default bridge to a user defined
// network because the default bridge is actually special versus any other
// docker network and lacks the emebdded DNS
//
// for now this also makes it easier for apps to join the same network
const fixedNetworkName = "kind"

// ensureNetwork checks if docker network by name exists, if not it creates it
func ensureNetwork(name string, ipFamily config.ClusterIPFamily) error {
	// TODO: the network might already exist and not have ipv6 ... :|
	// discussion: https://github.com/kubernetes-sigs/kind/pull/1508#discussion_r414594198
	out, err := exec.Output(exec.Command(
		"docker", "network", "ls",
		"--filter=name=^"+regexp.QuoteMeta(name)+"$",
		"--format={{.Name}}",
	))
	if err != nil {
		return err
	}
	// network already exists
	if string(out) == name+"\n" {
		return nil
	}
	if ipFamily == config.IPv6Family {
		return exec.Command("docker", "network", "create", "-d=bridge", "--ipv6", name).Run()
	}
	return exec.Command("docker", "network", "create", "-d=bridge", name).Run()
}
