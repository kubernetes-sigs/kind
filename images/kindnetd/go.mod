module sigs.k8s.io/kind/images/kindnetd

go 1.13

require (
	github.com/coreos/go-iptables v0.4.5
	github.com/pkg/errors v0.9.1
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.0-20200520041808-52d707b772fe // indirect
	golang.org/x/sys v0.0.0-20200724161237-0e2f3a69832c // indirect
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200724153422-f32512634ab7
)
