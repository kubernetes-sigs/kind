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

// Package runtime contains functions for getting runtime information.
package runtime

import (
	"os"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/log"
)

// GetDefault selected the default runtime from the environment override
func GetDefault(logger log.Logger) cluster.ProviderOption {
	switch p := os.Getenv("KIND_EXPERIMENTAL_PROVIDER"); p {
	case "":
		return nil
	case "podman":
		logger.Warn("using podman due to KIND_EXPERIMENTAL_PROVIDER")
		return cluster.ProviderWithPodman()
	case "docker":
		logger.Warn("using docker due to KIND_EXPERIMENTAL_PROVIDER")
		return cluster.ProviderWithDocker()
	case "nerdctl", "finch", "nerdctl.lima":
		logger.Warnf("using %s due to KIND_EXPERIMENTAL_PROVIDER", p)
		return cluster.ProviderWithNerdctl(p)
	case "swarm":
		hostsRaw := os.Getenv("KIND_HOSTS")
		var hosts []cluster.SwarmHost
		if hostsRaw != "" {
			h, err := cluster.ParseSwarmHosts(hostsRaw)
			if err != nil {
				logger.Warnf("invalid KIND_HOSTS %q: %v", hostsRaw, err)
				return nil
			}
			hosts = h
		}
		bootstrap := os.Getenv("KIND_BOOTSTRAP_SWARM") != ""
		if len(hosts) == 0 {
			logger.Warn("using swarm provider (host list will come from the `hosts:` block of --config)")
		} else {
			logger.Warnf("using swarm provider across %d host(s) due to KIND_EXPERIMENTAL_PROVIDER", len(hosts))
		}
		return cluster.ProviderWithSwarm(hosts, bootstrap)
	default:
		logger.Warnf("ignoring unknown value %q for KIND_EXPERIMENTAL_PROVIDER", p)
		return nil
	}
}
