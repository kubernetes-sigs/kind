/*
Copyright The Kubernetes Authors.

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

package cli

import (
	"os"

	"sigs.k8s.io/kind/pkg/cluster/constants"
)

// ClusterNameEnv is the environment variable used to override the default
// cluster name.
const ClusterNameEnv = "KIND_CLUSTER_NAME"

// NameFlagHelp is the shared help text for the --name flag.
const NameFlagHelp = "cluster name (or via KIND_CLUSTER_NAME)"

// NameFromEnv returns the cluster name from KIND_CLUSTER_NAME.
func NameFromEnv() string {
	return os.Getenv(ClusterNameEnv)
}

// DefaultName returns the effective default cluster name.
func DefaultName() string {
	if name := NameFromEnv(); name != "" {
		return name
	}
	return constants.DefaultClusterName
}
