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
		logger.Warn("using runtime \"podman\" due to KIND_EXPERIMENTAL_PROVIDER")
		return cluster.ProviderWithPodman()
	case "docker":
		logger.Warn("using runtime \"docker\" due to KIND_EXPERIMENTAL_PROVIDER")
		return cluster.ProviderWithDocker()
	default:
		logger.Warnf("ignoring unknown runtime %q from KIND_EXPERIMENTAL_PROVIDER", p)
		return nil
	}
}
