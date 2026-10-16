---
title: "Resource Requirements"
menu:
  main:
    parent: "user"
    identifier: "user-resource-requirements"
    weight: 3
description: |-
  Notes on the resources kind clusters consume and how to measure usage on a running cluster.
---

## Overview

kind runs Kubernetes node containers on the host's container runtime, so the
host's CPU, memory, and disk I/O budgets apply directly to the cluster.

There is no single recommended figure that holds across workloads. Real
resource usage is dominated by what you run on top of the cluster, the
Kubernetes version, and the CNI in use, so the most reliable approach is to
measure against your own scenario rather than aim at a fixed number.

## What Tends to Limit Small Hosts

On very constrained hosts the bottleneck is usually **disk I/O for etcd**
rather than CPU or memory. etcd is sensitive to fsync latency, so SSD-backed
storage and a host that is not heavily contended for I/O make the largest
difference for cluster stability.

If you primarily plan to **build kind node images** (for example, with
`kind build node-image` on macOS or Windows), the floor is higher and is
about VM resources rather than cluster resources; see
[Settings for Docker Desktop][docker-desktop-settings] in the quick start.

## Measuring Cluster Usage

kind labels every node container with `io.x-k8s.kind.cluster=<name>`. Combine
that label with the runtime's `stats` command to watch live CPU and memory
use for a specific cluster (`kind` by default).

For Docker:

```
docker ps --filter label=io.x-k8s.kind.cluster=kind --format '{{.Names}}'
docker stats $(docker ps --filter label=io.x-k8s.kind.cluster=kind --format '{{.Names}}')
```

For [Podman][podman]:

```
podman ps --filter label=io.x-k8s.kind.cluster=kind --format '{{.Names}}'
podman stats $(podman ps --filter label=io.x-k8s.kind.cluster=kind --format '{{.Names}}')
```

For [nerdctl][nerdctl]:

```
nerdctl ps --filter label=io.x-k8s.kind.cluster=kind --format '{{.Names}}'
nerdctl stats $(nerdctl ps --filter label=io.x-k8s.kind.cluster=kind --format '{{.Names}}')
```

For per-pod measurements, install [metrics-server] (not bundled by default)
and use `kubectl top nodes` / `kubectl top pods -A`.

[docker-desktop-settings]: /docs/user/quick-start/#settings-for-docker-desktop
[podman]: https://podman.io/
[nerdctl]: https://github.com/containerd/nerdctl
[metrics-server]: https://github.com/kubernetes-sigs/metrics-server
