---
title: kind
---
<p style="text-align: center; margin-top: 2em; margin-bottom: -.75em;"><img alt="kind" src="./logo/logo.png" width="300px" /></p>

[kind] is a tool for running local Kubernetes clusters using Docker container "nodes".  
kind was primarily designed for testing Kubernetes itself, but may be used for local development or CI.

If you have [go] 1.16+ and [docker] or [podman] installed `go install sigs.k8s.io/kind@{{< stableVersion >}} && kind create cluster` is all you need!

<img src="images/kind-create-cluster.png" />

kind consists of:

- Go [packages][packages] implementing [cluster creation][cluster package], [image build][build package], etc.
- A command line interface ([`kind`][kind cli]) built on these packages.
- Docker [image(s)][images] written to run systemd, Kubernetes, etc.
- [`kubetest`][kubetest] integration also built on these packages (WIP)

kind bootstraps each "node" with [kubeadm][kubeadm]. For more details see [the design documentation][design doc].

**NOTE**: kind is still a work in progress, see the [1.0 roadmap].

## Installation and usage

For more detailed instructions see [the user guide][user guide].

You can install kind with `go install sigs.k8s.io/kind@{{< stableVersion>}}` (for [go] [1.17+][go-supported]). This will put `kind` in
`$(go env GOPATH)/bin`. You may need to add that directory to your `$PATH` as
shown [here](https://golang.org/doc/code.html#GOPATH) if you encounter the error
`kind: command not found` after installation.

To use kind, you will also need to [install docker].  
Once you have docker running you can create a cluster with:

{{< codeFromInline lang="bash" >}}
kind create cluster
{{< /codeFromInline >}}

To delete your cluster use:

{{< codeFromInline lang="bash" >}}
kind delete cluster
{{< /codeFromInline >}}

<!--TODO(bentheelder): improve this part of the guide-->
To create a cluster from Kubernetes source:

- ensure that Kubernetes is cloned in `$(go env GOPATH)/src/k8s.io/kubernetes`
- build a node image and create a cluster with 

{{< codeFromInline lang="bash" >}}
kind build node-image
kind create cluster --image kindest/node:latest
{{< /codeFromInline >}}

Multi-node clusters and other advanced features may be configured with a config
file, for more usage see [the user guide][user guide] or run `kind [command] --help`

## Community

Please reach out for bugs, feature requests, and other issues!  
The maintainers of this project are reachable via:

- [Kubernetes Slack] in the [#kind] channel
- [filing an issue] against this repo
- The Kubernetes [SIG-Testing Mailing List]

Current maintainers are [@aojea] and [@BenTheElder] -- feel free to
reach out directly if you have any questions!

Pull Requests are very welcome!  
If you're planning a new feature, please file an issue to discuss first.

Check the [issue tracker] for `help wanted` issues if you're unsure where to
start, or feel free to reach out to discuss. 🙂

See also: our own [contributor guide] and the Kubernetes [community page]. 

## Why kind?

- kind supports multi-node (including HA) clusters
- kind supports building Kubernetes release builds from source
  - support for make / bash or docker, in addition to pre-published builds
- kind supports Linux, macOS and Windows
- kind is a [CNCF certified conformant Kubernetes installer](https://landscape.cncf.io/?selected=kind)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct].

<!--links-->
[kind]: https://sigs.k8s.io/kind
[go]: https://golang.org/
[go-supported]: https://golang.org/doc/devel/release.html#policy
[docker]: https://www.docker.com/
[podman]: https://podman.io/
[community page]: https://kubernetes.io/community/
[Kubernetes Code of Conduct]: https://github.com/kubernetes/community/blob/master/code-of-conduct.md
[Go Report Card Badge]: https://goreportcard.com/badge/sigs.k8s.io/kind
[Go Report Card]: https://goreportcard.com/report/sigs.k8s.io/kind
[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[packages]: https://github.com/kubernetes-sigs/kind/tree/main/pkg
[cluster package]: https://github.com/kubernetes-sigs/kind/tree/main/pkg/cluster
[build package]: https://github.com/kubernetes-sigs/kind/tree/main/pkg/build
[kind cli]: https://github.com/kubernetes-sigs/kind/tree/main/main.go
[images]: https://github.com/kubernetes-sigs/kind/tree/main/images
[kubetest]: https://github.com/kubernetes/test-infra/tree/master/kubetest
[kubeadm]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/
[design doc]: ./docs/design/initial
[user guide]: ./docs/user/quick-start
[SIG-Testing Mailing List]: https://groups.google.com/forum/#!forum/kubernetes-sig-testing
[issue tracker]: https://github.com/kubernetes-sigs/kind/issues
[filing an issue]: https://github.com/kubernetes-sigs/kind/issues/new
[Kubernetes Slack]: https://slack.k8s.io/
[#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
[1.0 roadmap]: /docs/contributing/1.0-roadmap
[install docker]: https://docs.docker.com/install/
[@BenTheElder]: https://github.com/BenTheElder
[@aojea]: https://github.com/aojea
[contributor guide]: /docs/contributing/getting-started
