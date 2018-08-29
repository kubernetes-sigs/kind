# TODO

A non-exhaustive list of tasks (in no-particular order) includes:
- [x] basic single "node" clusters
- [x] multiple clusters per host / named clusters
- [ ] user guides for common usage
  - [ ] user guides for more advanced usage
- [ ] preflight checks
- [ ] multi-node clusters
- [x] support for multiple kubernetes builds:
  - [x] bazel build from source
  - [x] docker / make build from source
  - [x] apt (upstream / official release packages)
    - [ ] support for selecting a non-default package version
- [ ] kubetest ingregration [WIP]
  - [ ] point existing ["dind"](https://github.com/kubernetes/test-infra/tree/master/dind) integration here once complete
- [ ] improved logging and error handling
- [ ] continuous integration
- [ ] publish pre-built images to a registry
- [ ] fake out all internals and unit test [WIP]
- [ ] pre-load images that are not from the build / possibly build more images
  - [ ] etcd
  - [ ] overlay network images?
- [ ] support multiple overlay networks
- [x] support advanced configuration via config file
  - [x] kubeadm config template override
- [ ] more advanced network configuration (not docker0)
- [ ] support for other CRI within the "node" containers (containerd, cri-o)
- [ ] switch from `exec.Command("docker", ...)` to the Docker client library
- [ ] log dumping functionality / cli commands
  - [ ] support audit logging
- [ ] cli command to list clusters

# Wishlist

Longer term / continually appealing items:

- Improved documentation
- Support for architectures / platforms other than linux / amd64 for the node images
- Support for client platforms other than docker on linux / docker for mac
- Less privileged containers or sharing a CRI via something like [containerd namespaces](https://github.com/containerd/containerd/blob/master/docs/namespaces.md), generally
 better isolation
- HA kubeadm / multiple control plane nodes
