module sigs.k8s.io/kind/images/kindnetd

go 1.13

require (
	github.com/coreos/go-iptables v0.6.0
	github.com/vishvananda/netlink v1.1.0
	// TODO: remove when no longer needed to be explicit to pickup CVE fix
	// indirect dep
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.24.1
	k8s.io/apimachinery v0.24.1
	k8s.io/client-go v0.24.1
	k8s.io/klog/v2 v2.60.1
)
