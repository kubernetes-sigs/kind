# Reduced_kind

A reduced re-implementation of [kind](https://kind.sigs.k8s.io/) targeting
**multi-host Kubernetes-in-Docker via Docker Swarm**.

## Goals

1. **Single provider** — Docker only (no Podman / Nerdctl).
2. **Multi-host** — nodes can live on different physical / virtual hosts.
3. **Reuse kind defaults** — same node image, same subnets, same ports, same
   labels.  Configurations written for kind should mostly Just Work.
4. **Simple** — drop edge cases that don't pay rent (rootless detection,
   userns-remap, IPv6 retry loops, etc.).

## Non-goals

- Replacing kind upstream.
- Building node images.  We consume `kindest/node:vX.Y.Z` produced by kind.
- Supporting `kind build node-image`.
- Running Kubernetes versions older than what the bundled `kindest/node` ships.

## Layout

```
Reduced_kind/
├── README.md          ← this file
├── DESIGN.md          ← multi-host via Docker Swarm — design doc (think only)
└── config/
    ├── types.go       ← cluster / node / networking schema
    └── defaults.go    ← all default values, gathered in one place
```

## Status

- [x] Project skeleton
- [x] Default config values extracted from kind
- [x] Multi-host design doc
- [ ] Implementation (deferred — see DESIGN.md)

## How this maps to kind

| kind path                                       | Reduced_kind equivalent          |
| ----------------------------------------------- | -------------------------------- |
| `pkg/apis/config/v1alpha4/types.go`             | `config/types.go` (subset)       |
| `pkg/apis/config/v1alpha4/default.go`           | `config/defaults.go`             |
| `pkg/apis/config/defaults/image.go`             | `config/defaults.go`             |
| `pkg/build/nodeimage/defaults.go`               | `config/defaults.go`             |
| `pkg/cluster/constants/constants.go`            | `config/defaults.go`             |
| `pkg/cluster/internal/providers/common/constants.go` | `config/defaults.go`        |
| `pkg/cluster/internal/providers/docker/constants.go` | `config/defaults.go`        |
