# TODO

NOTE: this document is slowly being phased out in favor of [GitHub issues](https://github.com/kubernetes-sigs/kind/issues),
and the [roadmap](./roadmap.md).

A non-exhaustive list of tasks (in no-particular order) includes:
- [x] basic single "node" clusters
- [x] multiple clusters per host / named clusters
- [x] user guide(s)
  - [x] more detailed user guides for common usage
  - [x] more detailed user guides for more advanced usage
- [ ] preflight checks [WIP]
- [ ] multi-node clusters
- [x] cli command to list clusters
- [x] support for multiple kubernetes builds:
  - [x] bazel build from source
  - [x] docker / make build from source
  - [x] apt (upstream / official release packages)
    - [ ] support for selecting a non-default package version
- [ ] kubetest ingregration [WIP]
- [ ] improved logging and error handling [WIP]
- [x] continuous integration
- [x] publish pre-built images to a registry
- [x] pre-load images that are not from the build / possibly build more images
  - [x] etcd
  - [x] coreDNS
  - [x] Kubernetes images for released builds
  - [ ] overlay network images?
- [ ] support multiple overlay networks
- [x] support advanced configuration via config file
  - [x] kubeadm config template override
  - [x] node lifecycle hooks
- [ ] more advanced network configuration (not docker0)
- [ ] support for other CRI within the "node" containers (containerd, cri-o)
- [ ] move all docker functionality into a common package (`pkg/docker`) [WIP]
 - [ ] potentially move this to using the docker client library
- [x] log dumping functionality / cli commands
  - [ ] support audit logging
- [ ] fake out all internals and unit test [WIP]
- [ ] support for local registries
- [ ] support for mounting extra directories

# Wishlist

Longer term / continually appealing items:

- Improved documentation
- Support for architectures / platforms other than linux / amd64 for the node images
- Support for client platforms other than docker on linux / docker for mac
- Less privileged containers or sharing a CRI via something like [containerd namespaces](https://github.com/containerd/containerd/blob/master/docs/namespaces.md), generally
 better isolation
- HA kubeadm / multiple control plane nodes
