<!--TODO(bentheelder): fill this in much more thoroughly-->
# `kind` - `k`ubernetes `in` `d`ocker

<!-- prow build badge, and go report card-->
<a href="https://prow.k8s.io/?job=ci-kind-build">
<img alt="Build Status" src="https://prow.k8s.io/badge.svg?jobs=ci-kind-build">
</a> <a href="https://goreportcard.com/report/sigs.k8s.io/kind"><img alt="Go Report Card" src="https://goreportcard.com/badge/sigs.k8s.io/kind" /></a>


`kind` is a tool for running local Kubernetes clusters using Docker container as "nodes".  
`kind` is primarily designed for testing Kubernetes 1.11+.
It initially targets the [conformance tests].

If you have [Go] and [Docker] installed, the following commands are all you need!

```console
go get -u sigs.k8s.io/kind
kind create cluster
export KUBECONFIG=$(kind get kubeconfig-path)
```

<img src="https://gist.githubusercontent.com/BenTheElder/621bc321fc6d9506fd936feb36d32dd0/raw/7fe14e9d0929cab428929ca6c501abc990c07359/kind-create-cluster.gif" alt="kind create cluster" />

`kind` consists of:
 - Go [packages][packages] implementing [cluster creation][cluster package], [image build][build package], etc.
 - A command line interface ([`kind`][kind cli]) built on these packages.
 - Docker [image(s)][images] written to run systemd, Kubernetes, etc.
 - [`kubetest`][kubetest] integration also built on these packages (currently WIP).

`kind` bootstraps each "node" with [kubeadm][kubeadm].
For more details see [the design documentation][design doc].  

**NOTE**: `kind` is still a work in progress.
See [docs/todo.md] for a list of pending tasks.

## Installation

You can install `kind` by running:

```console
go get -u sigs.k8s.io/kind
```

## Usage

To use `kind`, you need to [install Docker].
Once you have Docker running you can create a cluster by running:

```console
kind create cluster
```

To point at the newly created cluster, you can run:

```console
export KUBECONFIG=$(kind get kubeconfig-path)
```

To delete your cluster after you're done testing, use the following command:

```console
kind delete cluster
```

## Using a custom version of Kubernetes

To create a cluster from a specific Kubernetes version, you must ensure the following:

- That Kubernetes is cloned in `$(go env GOPATH)/src/k8s.io/kubernetes`.
- That the tag/commit you want to use is checked out in the above directory.

Then, you must build the base and node images (replacing `<base-image>` and `<node-image>` as desired):

```console
kind build base-image --image <base-image>
kind build node-image --base-image <base-image> --image <node-image>
```

After having performed these steps, you can launch a Kubernetes cluster by running:

```console
kind create cluster --image <node-image>
```

For more usage details check [the documentation][user guide] or run `kind [command] --help`

## Community, discussion, contribution, and support

Please reach out for bugs, feature requests, and other issues!
The maintainers of this project are reachable via:

- [Kubernetes Slack] in the [#kind] channel
- [Filing an issue] against this repo
- The Kubernetes [SIG-Testing Mailing List]

Current maintainers (approvers) are [@BenTheElder] and [@munnerz].

Pull Requests are very welcome!
See the [issue tracker] if you're unsure where to start, or feel free to reach out to discuss.

See also the Kubernetes [community page].

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct].

<!--links-->
[Go]: https://golang.org/
[Docker]: https://www.docker.com/
[community page]: http://kubernetes.io/community/
[Kubernetes Code of Conduct]: code-of-conduct.md
[Go Report Card Badge]: https://goreportcard.com/badge/sigs.k8s.io/kind
[Go Report Card]: https://goreportcard.com/report/sigs.k8s.io/kind
[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md
[todo]: ./docs/todo.md
[packages]: ./pkg
[cluster package]: ./pkg/cluster
[build package]: ./pkg/build
[kind cli]: ./main.go
[images]: ./images
[kubetest]: https://github.com/kubernetes/test-infra/tree/master/kubetest
[kubeadm]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/
[design doc]: ./docs/design/
[user guide]: ./docs/user/
[the docs]: ./docs
[SIG-Testing Mailing List]: https://groups.google.com/forum/#!forum/kubernetes-sig-testing
[issue tracker]: https://github.com/kubernetes-sigs/kind/issues
[filing an issue]: https://github.com/kubernetes-sigs/kind/issues/new
[Kubernetes Slack]: http://slack.k8s.io/
[#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
[docs/todo.md]: ./docs/todo.md
[install docker]: https://docs.docker.com/install/
[@BenTheElder]: https://github.com/BenTheElder
[@munnerz]: https://github.com/munnerz
