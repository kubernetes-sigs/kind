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

- SIG Testing
- SIG Cluster-Lifecycle
  - the kubeadm subproject
- Possibly SIG Release (mainly to provide easy access to alpha and beta tags)

**Covered Work**:

- limited workloads / e2e testing
- cluster bring-up (IE kubeadm)
- kubernetes build (and currently install, but that may be problematic for cross-platform [#166])
- node skew, client skew (kubectl / e2e)
- image publishing
- Kubernetes CI tooling and jobs
- ...

### P1: Support Testing Kubernetes Applications
---

**Stakeholders**: Various projects both inside & outside the Kubernetes Org.

- cert-manager
- cluster-api-provider-aws
- cluster-api-provider-azure
- ...

**Covered Work**:

Most of the necessary work should be covered under 
[P1: Support Testing Kubernetes Applications](#p1-support-testing-kubernetes-applications),
however there is some additional work.

- improve "kind as a library"
- ...

### P2: Provide Cheap Bootstrap Clusters for the Cluster-API 
---

**Stakeholders**:

- the cluster-api
- cluster-api-provider-foo developers

### P3: Extended Testing Not Covered Above
---

**Stakeholders**: 

- Indeterminate / many

Possibly supporting various things that we cannot reasonably test today including:

- "node" tests, e.g. reboot
- upgrades, downgrades
- anything depending on ingress
- anything depending on persistent storage / PVs
- testing the cluster-api proper with some sort of machine provisioning
- device plugin, e.g. GPU
- ...

Several of these make sense but are not possible with the current tooling and will require a reasonable amount of design and thought to do well. Some of them may not be solve-able in a good way, but are at least technologically feasible to explore.

### Out of Scope
---

Some things we can likely never cover in a reasonable way:

- cloud provider / CCM
- some of the node testing
- being a strange alternative library to "docker compose" etc.
- replacing [Phippy][phippy] ❤️ 🦒 ❤️
- ...


[#166]: https://github.com/kubernetes-sigs/kind/issues/166
[1.0 roadmap]: /docs/contributing/1.0-roadmap
[1.0 tracking milestone]: https://github.com/kubernetes-sigs/kind/milestone/2
[phippy]: https://phippy.io
