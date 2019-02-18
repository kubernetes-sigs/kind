---
title: "Project Scope"
menu:
  main:
    parent: "contributing"
    identifier: "project-scope"
    weight: 2
---
# Project Scope

This document outlines some scoping and major priorities for kind.

See also: the [1.0 roadmap], and the [1.0 tracking milestone].

## Priorities (from greatest to least)

### P-1: Bootstrapping the kind Project Itself
---

**Stakeholders**:

- kind maintainers
- kind contributors

**Covered Work**:

- Releases & tooling
- Automated image publishing
- Documentation bootstrapping (IE this site)
- Enough Kubernetes testing to test kind itself (Kubernetes Conformance tests)
- Setting up linters and other tools to verify quality
- Setting up a recurring subproject meeting

### P0: Support Testing Kubernetes
---

**Stakeholders**: 

- [SIG Testing][sigs]
- [SIG Cluster-Lifecycle][sigs]
  - the [kubeadm] subproject
- Possibly [SIG Release][sigs] (mainly to provide easy access to alpha and beta tags)

**Covered Work**:

- Limited workloads / [e2e testing][e2e]
- Cluster bring-up (IE [kubeadm])
- Kubernetes build (and currently install, but that may be problematic for cross-platform [#166])
- Node skew, client skew (kubectl / e2e versions)
- Image publishing
- Kubernetes CI tooling and [jobs][kubeadm-kind-job]
- Most everything in the [1.0 roadmap]
- ...

### P1: Support Testing Kubernetes Applications
---

**Stakeholders**: Various projects both inside & outside the Kubernetes Org.

- [cert-manager]
- [cluster-api-provider-aws]
- [cluster-api-provider-azure]
- ...

**Covered Work**:

Most of the necessary work should be covered under 
[P1: Support Testing Kubernetes Applications](#p1-support-testing-kubernetes-applications),
however there is some additional work.

- Improve "kind as a library"
  - better and more controllable logging
  - generally more control over output
  - example usage & documentation
  - better / tighter API contracts
- Most of the rest should be covered by improving "kind the binary" outlined above
- ...

### P2: Provide Cheap Bootstrap Clusters for the Cluster-API 
---

**Stakeholders**:

- various [cluster-api][cluster-api] [provider implementation][cluster-api provider implementations]
developers
- various [cluster-api][cluster-api] users

### P3: Extended Testing Not Covered Above
---

**Stakeholders**: 

- Indeterminate / many

Possibly supporting various things that we cannot reasonably test today including:

- "node" tests, e.g. reboot
- Upgrades, downgrades
- Anything depending on ingress
- Anything depending on persistent storage / PVs
- Testing the cluster-api proper with some sort of machine provisioning
- Device plugin, e.g. GPU
- ...

Several of these make sense but are not possible with the current tooling and will require a reasonable amount of design and thought to do well. Some of them may not be solve-able in a good way, but are at least technologically feasible to explore.

### Out of Scope
---

Some things we can likely never cover in a reasonable way:

- Cloud provider / [CCM]
- Some of the node testing (which portions exactly is currently unclear)
- Being an alternative to "docker compose" etc.
- Replacing [Phippy][phippy] ❤️ 🦒 ❤️
- ...


[#166]: https://github.com/kubernetes-sigs/kind/issues/166
[1.0 roadmap]: /docs/contributing/1.0-roadmap
[1.0 tracking milestone]: https://github.com/kubernetes-sigs/kind/milestone/2
[phippy]: https://phippy.io
[kubeadm]: https://github.com/kubernetes/kubeadm
[sigs]: https://github.com/kubernetes/community/blob/master/sig-list.md
[e2e]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/e2e-tests.md
[kubeadm-kind-job]: https://testgrid.k8s.io/sig-cluster-lifecycle-all#kubeadm-kind-master
[cert-manager]: https://github.com/jetstack/cert-manager
[cluster-api-provider-aws]: https://github.com/kubernetes-sigs/cluster-api-provider-aws
[cluster-api-provider-azure]: https://github.com/kubernetes-sigs/cluster-api-provider-azure
[cluster-api]: https://github.com/kubernetes-sigs/cluster-api
[cluster-api provider implementations]: https://github.com/kubernetes-sigs/cluster-api#provider-implementations
[CCM]: https://github.com/kubernetes/kubernetes/tree/master/cmd/cloud-controller-manager
