# kindnetd

`kindnetd` is a simple networking daemon with the following responsibilities:

- IP masquerade (of traffic leaving the nodes that is headed out of the cluster)
- Ensuring netlink routes to pod CIDRs via the host node IP for each
- Ensuring a simple CNI config based on the standard [ptp] / [host-local] [plugins] and the node's pod CIDR

kindnetd is based on [aojea/kindnet] which is in turn based on [leblancd/kube-v6-test].

We use this to implement KIND's standard CNI / cluster networking configuration.

## Building

cd to this directory on mac / linux with docker installed and run `make quick`.

To push an image run `make push`.

[ptp]: https://www.cni.dev/plugins/current/main/ptp/
[host-local]: https://www.cni.dev/plugins/current/ipam/host-local/
[plugins]: https://github.com/containernetworking/plugins
[aojea/kindnet]: https://github.com/aojea/kindnet
[leblancd/kube-v6-test]: https://github.com/leblancd/kube-v6-test/tree/master
