module sigs.k8s.io/kind/images/kindnetd

go 1.13

require (
	github.com/coreos/go-iptables v0.4.5
	github.com/pkg/errors v0.9.1
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.0-20200520041808-52d707b772fe // indirect
	golang.org/x/sys v0.0.0-20200923182605-d9f96fdee20d // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/klog/v2 v2.3.0
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
)
