// Package config defines the Reduced_kind cluster configuration schema.
//
// It mirrors kind's v1alpha4 schema with a couple of additions for multi-host
// (the Host field on Node and Networking.SwarmOverlay).  Fields that kind
// supports but Reduced_kind doesn't yet handle (containerd patches, etc.) are
// retained for forward-compatibility with kind YAML files.
package config

// Cluster is the top-level cluster configuration.
type Cluster struct {
	TypeMeta `yaml:",inline" json:",inline"`

	// Name of the cluster.  Defaults to DefaultClusterName ("kind").
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Nodes describes each container that will become part of the cluster.
	// If empty, defaults to a single control-plane node.
	Nodes []Node `yaml:"nodes,omitempty" json:"nodes,omitempty"`

	// Networking is the cluster-wide networking configuration.
	Networking Networking `yaml:"networking,omitempty" json:"networking,omitempty"`

	// FeatureGates are passed through to all Kubernetes components.
	FeatureGates map[string]bool `yaml:"featureGates,omitempty" json:"featureGates,omitempty"`

	// RuntimeConfig is passed to kube-apiserver via --runtime-config.
	RuntimeConfig map[string]string `yaml:"runtimeConfig,omitempty" json:"runtimeConfig,omitempty"`

	// KubeadmConfigPatches are RFC 7396 merge patches applied to the
	// generated kubeadm config.
	KubeadmConfigPatches []string `yaml:"kubeadmConfigPatches,omitempty" json:"kubeadmConfigPatches,omitempty"`
}

// TypeMeta carries Kind / APIVersion, like Kubernetes API objects.
type TypeMeta struct {
	Kind       string `yaml:"kind,omitempty" json:"kind,omitempty"`
	APIVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
}

// Node describes one container in the cluster.
type Node struct {
	// Role is one of "control-plane" or "worker".  Defaults to control-plane.
	Role NodeRole `yaml:"role,omitempty" json:"role,omitempty"`

	// Image is the node image to run.  Defaults to DefaultNodeImage.
	Image string `yaml:"image,omitempty" json:"image,omitempty"`

	// Labels are applied to the registered Kubernetes Node.
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	// ExtraMounts are bind-mounts from host to container.
	ExtraMounts []Mount `yaml:"extraMounts,omitempty" json:"extraMounts,omitempty"`

	// ExtraPortMappings exposes container ports on the host.
	ExtraPortMappings []PortMapping `yaml:"extraPortMappings,omitempty" json:"extraPortMappings,omitempty"`

	// Host is the Reduced_kind addition: which host this node should run on.
	//
	// Value is a docker context name (see `docker context ls`).  Empty means
	// "the current docker context", which preserves kind single-host behavior.
	//
	// In multi-host mode this names a Swarm worker / manager that the node
	// container will be created on via `docker --context=<Host> run ...`.
	Host string `yaml:"host,omitempty" json:"host,omitempty"`
}

// NodeRole is "control-plane" or "worker".
type NodeRole string

const (
	ControlPlaneRole NodeRole = "control-plane"
	WorkerRole       NodeRole = "worker"
)

// Networking is the cluster-wide network configuration.
type Networking struct {
	IPFamily         ClusterIPFamily `yaml:"ipFamily,omitempty" json:"ipFamily,omitempty"`
	APIServerPort    int32           `yaml:"apiServerPort,omitempty" json:"apiServerPort,omitempty"`
	APIServerAddress string          `yaml:"apiServerAddress,omitempty" json:"apiServerAddress,omitempty"`
	PodSubnet        string          `yaml:"podSubnet,omitempty" json:"podSubnet,omitempty"`
	ServiceSubnet    string          `yaml:"serviceSubnet,omitempty" json:"serviceSubnet,omitempty"`
	DisableDefaultCNI bool           `yaml:"disableDefaultCNI,omitempty" json:"disableDefaultCNI,omitempty"`
	KubeProxyMode    ProxyMode       `yaml:"kubeProxyMode,omitempty" json:"kubeProxyMode,omitempty"`

	// SwarmOverlay is the Reduced_kind addition: when true the provider
	// creates a Swarm overlay network spanning every host instead of a
	// single-host bridge network.
	//
	// Required when any Node.Host is non-empty.
	SwarmOverlay bool `yaml:"swarmOverlay,omitempty" json:"swarmOverlay,omitempty"`
}

// ClusterIPFamily picks the protocol used inside the cluster.
type ClusterIPFamily string

const (
	IPv4Family      ClusterIPFamily = "ipv4"
	IPv6Family      ClusterIPFamily = "ipv6"
	DualStackFamily ClusterIPFamily = "dual"
)

// ProxyMode is the kube-proxy data-plane.
type ProxyMode string

const (
	IPTablesProxyMode ProxyMode = "iptables"
	IPVSProxyMode     ProxyMode = "ipvs"
	NFTablesProxyMode ProxyMode = "nftables"
)

// Mount is a bind mount from host into the node container.
type Mount struct {
	ContainerPath  string `yaml:"containerPath,omitempty" json:"containerPath,omitempty"`
	HostPath       string `yaml:"hostPath,omitempty" json:"hostPath,omitempty"`
	Readonly       bool   `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`
	SelinuxRelabel bool   `yaml:"selinuxRelabel,omitempty" json:"selinuxRelabel,omitempty"`
}

// PortMapping publishes a container port to the host.
type PortMapping struct {
	ContainerPort int32  `yaml:"containerPort,omitempty" json:"containerPort,omitempty"`
	HostPort      int32  `yaml:"hostPort,omitempty" json:"hostPort,omitempty"`
	ListenAddress string `yaml:"listenAddress,omitempty" json:"listenAddress,omitempty"`
	Protocol      string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}
