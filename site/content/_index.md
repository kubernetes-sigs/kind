<p style="text-align: center; margin-top: 2em; margin-bottom: -.75em;"><img alt="kind" src="./logo/logo.png" width="300px" /></p>

[kind] is a tool for running local Kubernetes clusters using Docker container "nodes".  
kind is primarily designed for testing Kubernetes 1.11+, initially targeting the [conformance tests].

If you have [go] \([1.11+][go-supported]) and [docker] installed `GO111MODULE="on" go get sigs.k8s.io/kind@v0.5.0 && kind create cluster` is all you need!

<img src="https://gist.githubusercontent.com/BenTheElder/621bc321fc6d9506fd936feb36d32dd0/raw/13fe81c219e64b4917575c8988e06719c072c7f1/kind-demo.gif" alt="2x speed `kind create cluster` demo" />

kind consists of:

- Go [packages][packages] implementing [cluster creation][cluster package], [image build][build package], etc.
- A command line interface ([`kind`][kind cli]) built on these packages.
- Docker [image(s)][images] written to run systemd, Kubernetes, etc.
- [`kubetest`][kubetest] integration also built on these packages (WIP)

kind bootstraps each "node" with [kubeadm][kubeadm]. For more details see [the design documentation][design doc].

**NOTE**: kind is still a work in progress, see the [1.0 roadmap].

## Installation and usage

For more detailed instructions see [the user guide][user guide].

You can install kind with `go get sigs.k8s.io/kind`. This will put `kind` in
`$(go env GOPATH)/bin`. You may need to add that directory to your `$PATH` as
shown [here](https://golang.org/doc/code.html#GOPATH) if you encounter the error
`kind: command not found` after installation.

To use kind, you will also need to [install docker].  
Once you have docker running you can create a cluster with `kind create cluster`  
To delete your cluster use `kind delete cluster`

<!--TODO(bentheelder): improve this part of the guide-->
To create a cluster from Kubernetes source:

- ensure that Kubernetes is cloned in `$(go env GOPATH)/src/k8s.io/kubernetes`
- build a node image and create a cluster with `kind build node-image && kind create cluster --image kindest/node:latest`

Multi-node clusters and other advanced features may be configured with a config
file, for more usage see [the user guide][user guide] or run `kind [command] --help`

## Community, discussion, contribution, and support

Please reach out for bugs, feature requests, and other issues!  
The maintainers of this project are reachable via:

- [Kubernetes Slack] in the [#kind] channel
- [filing an issue] against this repo
- The Kubernetes [SIG-Testing Mailing List]

Current maintainers are [@BenTheElder] and [@munnerz] - feel free to
reach out if you have any questions!

Pull Requests are very welcome!  
See the [issue tracker] if you're unsure where to start, or feel free to reach out to discuss.

See also: our own [contributor guide] and the Kubernetes [community page]. 

## Why kind?

- kind supports multi-node (including HA) clusters
- kind supports building Kubernetes release builds from source
  - support for make / bash / docker, or bazel, in addition to pre-published builds.
- kind is written in go, and can be used as a library, has stable releases
- kind supports Windows in addition to MacOS and Linux
- kind is a [CNCF certified conformant Kubernetes installer](https://landscape.cncf.io/selected=kind)

## Alternatives

Some other open source projects with slightly different but overlapping use cases, features etc.

- https://github.com/bsycorp/kind
- https://github.com/ubuntu/microk8s
- https://github.com/kinvolk/kube-spawn
- https://github.com/kubernetes/minikube
- https://github.com/danderson/virtuakube
- https://github.com/kubernetes-sigs/kubeadm-dind-cluster

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct].

<!--links-->
[kind]: https://sigs.k8s.io/kind
[go]: https://golang.org/
[go-supported]: https://golang.org/doc/devel/release.html#policy
[docker]: https://www.docker.com/
[community page]: http://kubernetes.io/community/
[Kubernetes Code of Conduct]: code-of-conduct.md
[Go Report Card Badge]: https://goreportcard.com/badge/sigs.k8s.io/kind
[Go Report Card]: https://goreportcard.com/report/sigs.k8s.io/kind
[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[packages]: https://github.com/kubernetes-sigs/kind/tree/master/pkg
[cluster package]: https://github.com/kubernetes-sigs/kind/tree/master/pkg/cluster
[build package]: https://github.com/kubernetes-sigs/kind/tree/master/pkg/build
[kind cli]: https://github.com/kubernetes-sigs/kind/tree/master/main.go
[images]: https://github.com/kubernetes-sigs/kind/tree/master/images
[kubetest]: https://github.com/kubernetes/test-infra/tree/master/kubetest
[kubeadm]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/
[design doc]: ./docs/design/initial
[user guide]: ./docs/user/quick-start
[SIG-Testing Mailing List]: https://groups.google.com/forum/#!forum/kubernetes-sig-testing
[issue tracker]: https://github.com/kubernetes-sigs/kind/issues
[filing an issue]: https://github.com/kubernetes-sigs/kind/issues/new
[Kubernetes Slack]: http://slack.k8s.io/
[#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
[1.0 roadmap]: /docs/contributing/1.0-roadmap
[install docker]: https://docs.docker.com/install/
[@BenTheElder]: https://github.com/BenTheElder
[@munnerz]: https://github.com/munnerz
[contributor guide]: /docs/contributing/getting-started
