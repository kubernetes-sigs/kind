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

package cli

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kind/pkg/cluster/constants"
)

// GetClusterNameFlags get cluster name from flags and lookup
// KIND_CLUSTER_NAME env if flags not set. default value is kind
func GetClusterNameFlags(cmd *cobra.Command) string {
	nameFromFlags, err := cmd.Flags().GetString("name")
	if err != nil {
		log.Fatalf("error accessing flag name for command %s: %v", cmd.Name(), err)
	}

	if len(nameFromFlags) > 0 {
		return nameFromFlags
	} else if nameFromEnv := os.Getenv("KIND_CLUSTER_NAME"); len(nameFromEnv) > 0 {
		return nameFromEnv
	} else {
		return constants.DefaultClusterName
	}
}
