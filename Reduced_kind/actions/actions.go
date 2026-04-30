// Package actions is the post-provisioning pipeline.
//
// kind's pipeline has 7 actions; Reduced_kind keeps the 4 that are
// strictly required to bring up a Kubernetes cluster:
//
//	WriteKubeadmConfig  — render kubeadm.conf onto every node
//	KubeadmInit         — `kubeadm init` on the bootstrap CP node
//	InstallCNI          — apply the kindnetd manifest pre-baked into the image
//	KubeadmJoin         — `kubeadm join` for every other node
//
// The signatures match cluster.Action so they can be slotted into a
// pipeline passed to cluster.Create.
package actions

import (
	"fmt"
	"strings"
	"time"

	"reducedkind/cluster"
	"reducedkind/config"
)

// All returns the standard 4-step pipeline plus a final readiness wait.
func All() []cluster.Action {
	return []cluster.Action{
		WriteKubeadmConfig,
		KubeadmInit,
		InstallCNI,
		KubeadmJoin,
		WaitForReady,
	}
}

// WriteKubeadmConfig renders /kind/kubeadm.conf on every node.  Both init
// and join read this file via `kubeadm --config`.
func WriteKubeadmConfig(nodes []cluster.Node, cfg *config.Cluster, p cluster.Provider) error {
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return fmt.Errorf("no control-plane node")
	}
	cpIP, err := cp.IP()
	if err != nil {
		return err
	}
	for _, n := range nodes {
		conf := renderKubeadmConfig(cfg, n.Name(), cpIP, n.Role() == string(config.ControlPlaneRole))
		if err := n.WriteFile("/kind/kubeadm.conf", conf); err != nil {
			return err
		}
	}
	return nil
}

// KubeadmInit bootstraps the control plane on the first CP node.
func KubeadmInit(nodes []cluster.Node, _ *config.Cluster, _ cluster.Provider) error {
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return fmt.Errorf("no control-plane node")
	}
	out, err := cp.Exec("kubeadm", "init",
		"--config=/kind/kubeadm.conf",
		"--skip-token-print",
	)
	if err != nil {
		return fmt.Errorf("kubeadm init failed: %v\n%s", err, out)
	}

	// Single-node clusters: remove the control-plane taint so user pods
	// can schedule.  In multi-node clusters the workers handle that.
	if len(nodes) == 1 {
		_, _ = cp.Exec("kubectl",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"taint", "nodes", "--all",
			"node-role.kubernetes.io/control-plane-",
		)
	}
	return nil
}

// InstallCNI applies the kindnetd manifest that the node image ships with.
func InstallCNI(nodes []cluster.Node, _ *config.Cluster, _ cluster.Provider) error {
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return fmt.Errorf("no control-plane node")
	}
	_, err := cp.Exec("kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		"apply", "-f", "/kind/manifests/default-cni.yaml",
	)
	return err
}

// KubeadmJoin runs kubeadm join on every worker node.
func KubeadmJoin(nodes []cluster.Node, _ *config.Cluster, _ cluster.Provider) error {
	for _, n := range cluster.FilterByRole(nodes, string(config.WorkerRole)) {
		out, err := n.Exec("kubeadm", "join", "--config=/kind/kubeadm.conf")
		if err != nil {
			return fmt.Errorf("kubeadm join %s failed: %v\n%s", n.Name(), err, out)
		}
	}
	return nil
}

// WaitForReady polls the API server until every node is Ready or 2 minutes
// elapse.  Returns nil on timeout — the cluster may still be usable.
func WaitForReady(nodes []cluster.Node, _ *config.Cluster, _ cluster.Provider) error {
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return fmt.Errorf("no control-plane node")
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		out, err := cp.Exec("kubectl",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"get", "nodes",
			"-o=jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}",
		)
		if err == nil && len(out) > 0 && !strings.Contains(string(out), "False") {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil // soft-fail
}

// renderKubeadmConfig is a deliberately tiny kubeadm config template.
// The real kind one is ~120 lines and includes patches, dual-stack, etc.
func renderKubeadmConfig(cfg *config.Cluster, nodeName, cpIP string, isCP bool) string {
	endpoint := fmt.Sprintf("%s:6443", cpIP)
	if isCP {
		return fmt.Sprintf(`apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
controlPlaneEndpoint: %s
networking:
  podSubnet: %s
  serviceSubnet: %s
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
nodeRegistration:
  name: %s
  criSocket: unix:///run/containerd/containerd.sock
localAPIEndpoint:
  advertiseAddress: %s
  bindPort: 6443
bootstrapTokens:
- token: abcdef.0123456789abcdef
`, endpoint, cfg.Networking.PodSubnet, cfg.Networking.ServiceSubnet, nodeName, cpIP)
	}
	return fmt.Sprintf(`apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
nodeRegistration:
  name: %s
  criSocket: unix:///run/containerd/containerd.sock
discovery:
  bootstrapToken:
    apiServerEndpoint: %s
    token: abcdef.0123456789abcdef
    unsafeSkipCAVerification: true
`, nodeName, endpoint)
}
