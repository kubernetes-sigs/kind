<!--TODO(bentheelder): fill this in much more thoroughly-->
# `kind` - **K**ubernetes **IN** **D**ocker

## WARNING: `kind` is still a work in progress! See [docs/todo.md][todo]

`kind` is a toolset for running local Kubernetes clusters using Docker container "nodes".
`kind` is designed to be suitable for testing Kubernetes, initially targeting the conformance suite.

It consists of:
 - Go [packages][packages] implementing [cluster creation][cluster package], [image build][build package], etc.
 - A command line interface ([`kind`][kind cli]) built on these packages.
 - Docker [image(s)][images] written to run systemd, Kubernetes, etc.
 - [`kubetest`][kubetest] integration also built on these packages (WIP)

Kind bootstraps each "node" with [kubeadm][kubeadm].

For more details see [the design documentation][design doc].

## Building

You can install `kind` with `go install k8s.io/test-infra/kind`.

## Usage

`kind create` will create a cluster.

`kind delete` will delete a cluster.

For more usage, run `kind --help` or `kind [command] --help`.

## Advanced

`kind build base` will build the base image.

`kind build node` will build the node image.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

<!--links-->
[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[todo]: ./docs/todo.md
[packages]: ./pkg
[cluster package]: ./pkg/cluster
[build package]: ./pkg/build
[kind cli]: ./main.go
[images]: ./images
[kubetest]: https://github.com/kubernetes/test-infra/tree/master/kubetest
[kubeadm]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/
[design doc]: ./docs/design.md
