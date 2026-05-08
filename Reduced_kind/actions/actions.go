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
	"bytes"
	"fmt"
	"strings"
	"text/template"
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
//
// kubeadm init normally takes 60–120s and is silent until done.  We use the
// streaming Exec variant (when available) plus --v=6 so the user sees
// progress instead of an apparent hang.
func KubeadmInit(nodes []cluster.Node, _ *config.Cluster, _ cluster.Provider) error {
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return fmt.Errorf("no control-plane node")
	}

	args := []string{"init", "--config=/kind/kubeadm.conf", "--skip-token-print", "--v=6"}

	// Prefer streaming output if the concrete Node supports it.
	type streamer interface {
		ExecStream(cmd string, args ...string) error
	}
	if s, ok := cp.(streamer); ok {
		if err := s.ExecStream("kubeadm", args...); err != nil {
			return fmt.Errorf("kubeadm init failed: %w", err)
		}
	} else {
		out, err := cp.Exec("kubeadm", args...)
		if err != nil {
			return fmt.Errorf("kubeadm init failed: %v\n%s", err, out)
		}
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
//
// The manifest in /kind/manifests/default-cni.yaml contains Go template
// placeholders (notably {{ .PodSubnet }}); kind renders these before
// apply.  We do the same.
func InstallCNI(nodes []cluster.Node, cfg *config.Cluster, _ cluster.Provider) error {
	cp := cluster.FindByRole(nodes, string(config.ControlPlaneRole))
	if cp == nil {
		return fmt.Errorf("no control-plane node")
	}

	// Read the raw template from the node.
	raw, err := cp.Exec("cat", "/kind/manifests/default-cni.yaml")
	if err != nil {
		return fmt.Errorf("read CNI manifest: %v\n%s", err, raw)
	}

	// Render Go template -> final YAML.
	t, err := template.New("cni").Parse(string(raw))
	if err != nil {
		return fmt.Errorf("parse CNI manifest: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, struct{ PodSubnet string }{
		PodSubnet: cfg.Networking.PodSubnet,
	}); err != nil {
		return fmt.Errorf("render CNI manifest: %w", err)
	}

	// Drop the rendered version next to the original and apply that.
	if err := cp.WriteFile("/kind/manifests/default-cni.rendered.yaml", buf.String()); err != nil {
		return fmt.Errorf("write rendered CNI manifest: %w", err)
	}
	out, err := cp.Exec("kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		"apply", "-f", "/kind/manifests/default-cni.rendered.yaml",
	)
	if err != nil {
		return fmt.Errorf("install CNI failed: %v\n%s", err, out)
	}
	return nil
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
//
// Two extra blocks are emitted for environment compatibility:
//   - KubeletConfiguration with failSwapOn=false so kubelet starts on
//     hosts that have swap (notably WSL2).
//   - KubeProxyConfiguration with conntrack.maxPerCore=0 / min=0 so
//     kube-proxy doesn't try to write /proc/sys/net/netfilter/
//     nf_conntrack_max (read-only on Docker Desktop / WSL2).
func renderKubeadmConfig(cfg *config.Cluster, nodeName, cpIP string, isCP bool) string {
	endpoint := fmt.Sprintf("%s:6443", cpIP)
	kubeletCfg := `---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
failSwapOn: false
cgroupDriver: systemd
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
conntrack:
  maxPerCore: 0
  min: 0
`
	if isCP {
		return fmt.Sprintf(`apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
controlPlaneEndpoint: %s
networking:
  podSubnet: %s
  serviceSubnet: %s
---
apiVersion: kubeadm.k8s.io/v1beta4
kind: InitConfiguration
nodeRegistration:
  name: %s
  criSocket: unix:///run/containerd/containerd.sock
localAPIEndpoint:
  advertiseAddress: %s
  bindPort: 6443
bootstrapTokens:
- token: abcdef.0123456789abcdef
%s`, endpoint, cfg.Networking.PodSubnet, cfg.Networking.ServiceSubnet, nodeName, cpIP, kubeletCfg)
	}
	return fmt.Sprintf(`apiVersion: kubeadm.k8s.io/v1beta4
kind: JoinConfiguration
nodeRegistration:
  name: %s
  criSocket: unix:///run/containerd/containerd.sock
discovery:
  bootstrapToken:
    apiServerEndpoint: %s
    token: abcdef.0123456789abcdef
    unsafeSkipCAVerification: true
%s`, nodeName, endpoint, kubeletCfg)
}
