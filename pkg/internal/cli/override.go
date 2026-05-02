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
