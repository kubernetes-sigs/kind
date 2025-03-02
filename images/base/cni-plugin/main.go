/*
Copyright 2025 The Kubernetes Authors.

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

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/040"
	"github.com/containernetworking/cni/pkg/version"

	"k8s.io/utils/ptr"
)

// Only implement the minimum functionality defined by CNI required Kubernetes use cases.
// Uses a sqlite file to provide an API for dynamic configuration.
// xref: https://gist.github.com/aojea/571c29f1b35e5c411f8297a47227d39d

const (
	pluginName    = "cni-kindnet"
	hostPortMapv4 = "hostport-map-v4"
	hostPortMapv6 = "hostport-map-v6"
	// containerd hardcodes this value
	// https://github.com/containerd/containerd/blob/23500b8015c6f5c624ec630fd1377a990e9eccfb/internal/cri/server/helpers.go#L68
	defaultInterface = "eth0"
	// cniConfigPath is where kindnetd will write the computed CNI config
	cniConfigPath = "/etc/cni/net.d"

	dbFile = "cni.db"

	ipamRangesTable = `
CREATE TABLE IF NOT EXISTS ipam_ranges (
  id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for the IP range
  subnet TEXT NOT NULL,                 -- Subnet in CIDR notation (e.g., "10.244.0.0/16")
  description TEXT                      -- Optional description of the IP range
);	
`
	podsTable = `
CREATE TABLE IF NOT EXISTS pods (
  container_id TEXT PRIMARY KEY,     -- ID of the pod Sandbox on the container runtime
  name TEXT,                -- Kubernetes name of the pod
  namespace TEXT,           -- Kubernetes namespace of the pod
  uid TEXT,          			 -- Kubernetes UID of the pod
  netns TEXT NOT NULL,               -- Network namespace path of the pod
  ip_address_v4 TEXT,       -- IPv4 address assigned to the pod
  ip_address_v6 TEXT,       -- IPv6 address assigned to the pod
  ip_gateway_v4 TEXT,       -- IPv4 gateway assigned to the pod
  ip_gateway_v6 TEXT,       -- IPv6 gateway assigned to the pod
  interface_name TEXT NOT NULL,      -- Name of the network interface of the pod in the host
	interface_mtu INTEGER,             -- Interface mtu
  creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp of pod creation
	UNIQUE (ip_address_v4),           -- Unique constraint for IPv4 address
  UNIQUE (ip_address_v6)            -- Unique constraint for IPv6 address
);	
`

	portmapsTable = `
CREATE TABLE IF NOT EXISTS portmap_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    container_id TEXT NOT NULL,
    host_ip TEXT NOT NULL,
    host_port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    container_ip TEXT NOT NULL,
    container_port INTEGER NOT NULL,
    FOREIGN KEY (container_id) REFERENCES pods(container_id) ON DELETE CASCADE,
    UNIQUE (host_ip, host_port, protocol) -- Unique constraint
);
`
)

var (
	db     *sql.DB
	logger *log.Logger

	// ref: https://www.pathname.com/fhs/pub/fhs-2.3.html#THEVARHIERARCHY
	// injected for testing
	dbDir = "/var/lib/cni-kindnet"
)

func start() error {
	err := os.MkdirAll(dbDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("can not create application directory on %s %v", dbDir, err)
	}
	dbPath := filepath.Join(dbDir, dbFile)

	db, err = sql.Open("sqlite3", dbPath+"?_busy_timeout=1000")
	if err != nil {
		return fmt.Errorf("can not open cni database %s : %v", dbPath, err)
	}

	// Enable foreign key support
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("could not enable foreign keys: %v", err)
	}

	// WAL has better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("could not enable foreign keys: %v", err)
	}

	_, err = db.Exec(podsTable)
	if err != nil {
		return fmt.Errorf("can not create pods table on cni database %s : %v", dbPath, err)
	}

	_, err = db.Exec(portmapsTable)
	if err != nil {
		return fmt.Errorf("can not create portmaps table on cni database %s : %v", dbPath, err)
	}

	_, err = db.Exec(ipamRangesTable)
	if err != nil {
		return fmt.Errorf("can not create ipam ranges table on cni database %s : %v", dbPath, err)
	}
	// Create a new logger that writes to the file
	if logFile := os.Getenv("CNI_LOG_FILE"); logFile != "" {
		// Open the log file in append mode, create it if it doesn't exist
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("can not open log file %s : %v", logFile, err)
		}
		logger = log.New(file, "", log.LstdFlags)
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}
	return nil
}

func main() {
	err := start()
	if err != nil {
		log.Fatalf("failed to initialize plugin: %v", err)
	}

	skel.PluginMainFuncs(skel.CNIFuncs{
		Add:   cmdAdd,
		Del:   cmdDel,
		Check: cmdCheck,
	},
		version.PluginSupports("0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0"),
		"CNI plugin "+pluginName,
	)
}

// K8sArgs is the valid CNI_ARGS used for Kubernetes
// The field names need to match exact keys in containerd args for unmarshalling
// https://github.com/containerd/containerd/blob/ced9b18c231a28990617bc0a4b8ce2e81ee2ffa1/pkg/cri/server/sandbox_run.go#L526-L532
type K8sArgs struct {
	types.CommonArgs
	K8S_POD_NAME               types.UnmarshallableString // nolint: revive, stylecheck
	K8S_POD_NAMESPACE          types.UnmarshallableString // nolint: revive, stylecheck
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString // nolint: revive, stylecheck
	K8S_POD_UID                types.UnmarshallableString // nolint: revive, stylecheck
}

// PortMapEntry corresponds to a single entry in the port_mappings argument,
// see CNI CONVENTIONS.md
type PortMapEntry struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"hostIP"`
}

type KindnetConf struct {
	types.NetConf
	Ranges        []string `json:"ranges,omitempty"`
	RuntimeConfig struct {
		PortMaps []PortMapEntry `json:"portMappings,omitempty"`
	} `json:"runtimeConfig,omitempty"`
}

// NetworkConfig of the Pods simplified but the kubernetes use cases
// additional IPs and more complex configuration MUST be handled via
// DRA and NRI that provide better mechanism to handle those, specially
// since they are stateful and contain hooks and synchronization mechanisms
// with the Pod lifecucle.
type NetworkConfig struct {
	ContainerID string // unique identifies based on the Pod Sandbox
	// Kubernetes metadata
	Name      string
	Namespace string
	UID       string
	// network namespace
	NetNS         string
	InterfaceName string // name on the host, inside the container is always eth0
	// IPv4 configuration
	IPv4 net.IP
	GWv4 net.IP
	// IPv6 configuration
	IPv6 net.IP
	GWv6 net.IP
	// MTU
	MTU int
	// Portmaps
	PortMaps []PortMapConfig
}

type PortMapConfig struct {
	HostIP        string
	HostPort      int
	Protocol      string
	ContainerIP   string
	ContainerPort int
}

func deleteNetworkConfig(containerID string) error {
	if containerID == "" {
		return fmt.Errorf("id empty")
	}
	_, err := db.Exec("DELETE FROM pods WHERE container_id = ?", containerID)
	if err != nil {
		return fmt.Errorf("error deleting pod entry: %w", err)
	}
	return nil
}

func (n *NetworkConfig) ToCNIResult() current.Result {
	result := current.Result{
		CNIVersion: "0.4.0", // implement the minimum necessary to work in kubernetes
		IPs:        nil,     // Container runtimes in kubernetes only care about this field
		Interfaces: []*current.Interface{
			{Name: defaultInterface},
		},
	}
	// Kubelet will reorder the IPs families to match the cluster setup
	// Here we return IPv4 first always as it the most common setup.
	if n.IPv4 != nil {
		result.IPs = append(result.IPs,
			&current.IPConfig{
				Version:   "4",
				Interface: ptr.To(0), // there is only one interface
				Address:   net.IPNet{IP: n.IPv4, Mask: net.CIDRMask(32, 32)},
				Gateway:   n.GWv4,
			},
		)
	}
	if n.IPv6 != nil {
		result.IPs = append(result.IPs,
			&current.IPConfig{
				Version:   "6",
				Interface: ptr.To(0), // there is only one interface
				Address:   net.IPNet{IP: n.IPv6, Mask: net.CIDRMask(128, 128)},
				Gateway:   n.GWv6,
			},
		)
	}
	return result
}

func cmdAdd(args *skel.CmdArgs) (err error) {
	if args.Netns == "" {
		return nil
	}

	// avoid to persist data if the connection didn't succeed
	// so we don't rely on the CNI DEL to clean up, portmaps
	// will be reconciled so no need to worry about them.
	defer func() {
		if err != nil {
			_ = deleteNetworkConfig(args.ContainerID)
		}
	}()

	networkConfig, err := newNetworkConfig(args)
	if err != nil {
		return fmt.Errorf("no network configuration available: %v", err)
	}

	// retry three times to make this more resilient since is cheaper
	// to retry the interface creation here than roundtripping over
	// the CNI ADD and CNI DEL
	var interfaceCreationError error
	for i := 0; i < 3; i++ {
		interfaceCreationError = createPodInterface(networkConfig)
		if err != nil {
			deletePodInterface(defaultInterface, args.Netns)
		} else {
			break
		}
	}
	if interfaceCreationError != nil {
		return fmt.Errorf("fail to create veth interface: %v", interfaceCreationError)
	}
	if len(networkConfig.PortMaps) > 0 {
		err = reconcilePortMaps()
		if err != nil {
			return fmt.Errorf("fail to reconcile portmaps: %v", err)
		}
		// Delete conntrack entries for UDP to avoid conntrack blackholing traffic
		// due to stale connections. We do that after the dataplane rules are set, so
		// the new traffic uses them. Failures are informative only.
		if err := deletePortmapStaleConnections(networkConfig.PortMaps); err != nil {
			logger.Printf("failed to delete stale UDP conntrack entries for %s: %v", networkConfig.IPv4.String(), err)
		}
	}

	cniResult := networkConfig.ToCNIResult()
	return cniResult.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	if args.Netns == "" {
		return nil
	}
	conf := KindnetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	err := deletePodInterface(defaultInterface, args.Netns)
	if err != nil {
		return fmt.Errorf("fail to delete veth interface: %v", err)
	}

	err = deleteNetworkConfig(args.ContainerID)
	if err != nil {
		return fmt.Errorf("network configuration can not be deleted: %v", err)
	}

	if len(conf.RuntimeConfig.PortMaps) > 0 {
		err = reconcilePortMaps()
		if err != nil {
			return fmt.Errorf("fail to reconcile portmaps: %v", err)
		}
	}

	return nil
}

func cmdCheck(args *skel.CmdArgs) (err error) {
	defer db.Close()

	return nil
}
