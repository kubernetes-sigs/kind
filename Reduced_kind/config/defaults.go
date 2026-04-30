package config

// All default values used by Reduced_kind, gathered in one file.
//
// Values are sourced from the matching constants/defaults in kind:
//   - pkg/apis/config/defaults/image.go
//   - pkg/apis/config/v1alpha4/default.go
//   - pkg/build/nodeimage/defaults.go
//   - pkg/cluster/constants/constants.go
//   - pkg/cluster/internal/providers/common/constants.go
//   - pkg/cluster/internal/providers/docker/constants.go
//
// When upgrading kind, refresh these values from the same files.

// --- Images -----------------------------------------------------------------

const (
	// DefaultNodeImage is the node image to run when none is specified on a
	// Node.  Mirrors pkg/apis/config/defaults/image.go (Image).
	DefaultNodeImage = "kindest/node:v1.35.1@sha256:05d7bcdefbda08b4e038f644c4df690cdac3fba8b06f8289f30e10026720a1ab"

	// DefaultBaseImage is the base used when building a node image.  We
	// don't build images today, but we keep the constant for reference.
	// Mirrors pkg/build/nodeimage/defaults.go (DefaultBaseImage).
	DefaultBaseImage = "docker.io/kindest/base:v20260214-ea8e5717"

	// DefaultBuildImageTag is the default tag a freshly-built node image
	// would carry.  Mirrors pkg/build/nodeimage/defaults.go (DefaultImage).
	DefaultBuildImageTag = "kindest/node:latest"
)

// --- Cluster identity -------------------------------------------------------

const (
	// DefaultClusterName is the default cluster context name.
	// Mirrors pkg/cluster/constants/constants.go (DefaultClusterName).
	DefaultClusterName = "kind"

	// DefaultNetworkName is the docker network all nodes are attached to.
	// Mirrors pkg/cluster/internal/providers/docker/network.go (fixedNetworkName).
	DefaultNetworkName = "kind"
)

// --- Container labels (used as a database, see kind/docker/constants.go) ----

const (
	ClusterLabelKey  = "io.x-k8s.kind.cluster"
	NodeRoleLabelKey = "io.x-k8s.kind.role"
)

// --- Node roles -------------------------------------------------------------
//
// Mirrors pkg/cluster/constants/constants.go.

const (
	ControlPlaneNodeRoleValue         = "control-plane"
	WorkerNodeRoleValue               = "worker"
	ExternalLoadBalancerNodeRoleValue = "external-load-balancer"
	ExternalEtcdNodeRoleValue         = "external-etcd"
)

// --- Ports ------------------------------------------------------------------

const (
	// APIServerInternalPort is the port the Kubernetes API server listens
	// on inside each control-plane node container.
	// Mirrors pkg/cluster/internal/providers/common/constants.go.
	APIServerInternalPort = 6443
)

// --- Networking defaults ----------------------------------------------------
//
// Mirrors pkg/apis/config/v1alpha4/default.go (SetDefaultsCluster).

const (
	DefaultIPFamily = IPv4Family

	DefaultAPIServerAddressIPv4 = "127.0.0.1"
	DefaultAPIServerAddressIPv6 = "::1"

	DefaultPodSubnetIPv4      = "10.244.0.0/16"
	DefaultPodSubnetIPv6      = "fd00:10:244::/56"
	DefaultPodSubnetDualStack = "10.244.0.0/16,fd00:10:244::/56"

	DefaultServiceSubnetIPv4      = "10.96.0.0/16"
	DefaultServiceSubnetIPv6      = "fd00:10:96::/112"
	DefaultServiceSubnetDualStack = "10.96.0.0/16,fd00:10:96::/112"

	DefaultKubeProxyMode = IPTablesProxyMode
)

// --- Naming -----------------------------------------------------------------
//
// Mirrors pkg/cluster/internal/providers/common/namer.go.  First node in a
// role is unsuffixed; subsequent ones get "2", "3", ...

// SetDefaultsCluster fills in unset fields with default values.  Mirrors
// pkg/apis/config/v1alpha4/default.go.
func SetDefaultsCluster(c *Cluster) {
	if c.Name == "" {
		c.Name = DefaultClusterName
	}
	if len(c.Nodes) == 0 {
		c.Nodes = []Node{{Role: ControlPlaneRole, Image: DefaultNodeImage}}
	}
	for i := range c.Nodes {
		setDefaultsNode(&c.Nodes[i])
	}

	n := &c.Networking
	if n.IPFamily == "" {
		n.IPFamily = DefaultIPFamily
	}
	if n.APIServerAddress == "" {
		switch n.IPFamily {
		case IPv6Family:
			n.APIServerAddress = DefaultAPIServerAddressIPv6
		default:
			n.APIServerAddress = DefaultAPIServerAddressIPv4
		}
	}
	if n.PodSubnet == "" {
		switch n.IPFamily {
		case IPv6Family:
			n.PodSubnet = DefaultPodSubnetIPv6
		case DualStackFamily:
			n.PodSubnet = DefaultPodSubnetDualStack
		default:
			n.PodSubnet = DefaultPodSubnetIPv4
		}
	}
	if n.ServiceSubnet == "" {
		switch n.IPFamily {
		case IPv6Family:
			n.ServiceSubnet = DefaultServiceSubnetIPv6
		case DualStackFamily:
			n.ServiceSubnet = DefaultServiceSubnetDualStack
		default:
			n.ServiceSubnet = DefaultServiceSubnetIPv4
		}
	}
	if n.KubeProxyMode == "" {
		n.KubeProxyMode = DefaultKubeProxyMode
	}

	// Reduced_kind addition: any Node with Host set requires a Swarm overlay.
	for _, node := range c.Nodes {
		if node.Host != "" {
			n.SwarmOverlay = true
			break
		}
	}
}

func setDefaultsNode(n *Node) {
	if n.Role == "" {
		n.Role = ControlPlaneRole
	}
	if n.Image == "" {
		n.Image = DefaultNodeImage
	}
}
