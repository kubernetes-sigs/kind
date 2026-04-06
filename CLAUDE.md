# CLAUDE.md

<!-- For Claude Code setup guidance: https://code.claude.com/docs -->

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **cloud-provisioner**, a fork of the [kind](https://kind.sigs.k8s.io/) tool extended for enterprise multi-cloud Kubernetes cluster deployments (AWS/EKS, Azure/AKS, GCP/GKE). It uses Cluster API (CAPX) under the hood and supports KEOS cluster management.

## Build & Development Commands

### Prerequisites

The required Go version is defined in two files (keep them in sync when upgrading):

| File | Used by |
|------|---------|
| `.go-version` | Makefile via `hack/build/setup-go.sh` (gimme) |
| `.tool-versions` | asdf (`golang` entry) |

Check current required version:
```bash
cat .go-version       # e.g. 1.25.0
cat .tool-versions    # e.g. golang 1.25.8
```

### Standard Build

```bash
make clean build
make install INSTALL_DIR=$HOME/.local/bin   # or any directory in your PATH
```

The Makefile automatically resolves the Go toolchain via `hack/build/setup-go.sh` using
[gimme](hack/third_party/gimme/gimme) and `GOTOOLCHAIN=auto`. No manual `go` setup is needed.

**Verify the installed binary:**
```bash
which cloud-provisioner
cloud-provisioner version
```

**Run a single test:**
```bash
go test -v ./pkg/path/to/package/... -run TestName
```

**Key build env vars:** `GO111MODULE=on`, `CGO_ENABLED=0` (static binaries), `GOTOOLCHAIN=auto`

**Non-obvious make targets:**
```bash
make generate   # Regenerate DeepCopy code after API type changes (pkg/apis/config/v1alpha4/)
make update     # Run all code generation + gofmt
make verify     # Run all checks: lint + generated code + shellcheck
```

> Build troubleshooting (gimme errors, GOTOOLCHAIN issues) → `.claude/rules/build.md`

## Architecture

The codebase has several distinct layers:

### CLI Layer (`pkg/cmd/kind/`)
Cobra-based command structure. Root command is `cloud-provisioner` with sub-commands: `build`, `create`, `delete`, `export`, `get`, `load`, `completion`, `version`.

### API Configuration (`pkg/apis/config/v1alpha4/`)
Kubernetes-style YAML cluster definitions with generated DeepCopy code. Changes to types here require `make generate`.

### Cluster Management (`pkg/cluster/`)
Core cluster lifecycle (create, delete, get nodes). `provider.go` is the main entry point with a provider abstraction for Docker/Podman backends.

### Creation Actions (`pkg/cluster/internal/create/actions/`)
Each subdirectory is a discrete step in cluster creation: `kubeadminit`, `kubeadmjoin`, `installcni`, `installstorage`, `loadbalancer`, `createworker`, `waitforready`. The `createworker/` package is the most complex, handling cloud-provider-specific worker node bootstrapping.

### Common Utilities (`pkg/commons/`)
Shared types across the codebase: `ClusterConfig`, `KeosCluster`, CAPX versions, chart definitions, credential handling. `cluster.go` defines the core data structures.

### Infrastructure Providers (`pkg/cluster/internal/providers/`)
Docker and Podman provider implementations. Cloud-provider integrations (AWS, Azure, GCP) live in `createworker/`.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/commons/cluster.go` | Core data structures: `ClusterConfig`, `KeosCluster`, CAPX provider versions |
| `pkg/cluster/provider.go` | Main provider interface for cluster lifecycle |
| `pkg/cluster/internal/create/actions/createworker/createworker.go` | Worker node creation orchestration |
| `pkg/cluster/internal/create/actions/createworker/provider.go` | Cloud-provider-specific worker logic |
| `pkg/cluster/internal/create/actions/createworker/keosinstaller.go` | Generates and rotates `keos.yaml` descriptor; writes `aws_central_ecr_override.yml` when ECR central is enabled |
| `pkg/apis/config/v1alpha4/types.go` | Cluster configuration API types |

## PR and Branch Conventions

See `CONTRIBUTING.md` for the full PR workflow and label reference.
- Branch naming follows Jira ticket IDs (e.g., `PLT-3091`)

## Cloud Provider Structure

Cloud-provider logic is split across:
- `pkg/cluster/internal/create/actions/createworker/provider.go` — provider interface and dispatch
- `pkg/commons/` — credential types per provider (AWS, Azure, GCP)
- `docs/aws/`, `docs/azure/`, `docs/gcp/` — required IAM permissions per provider

The tool uses CAPX (Cluster API Provider X) with versions tracked in `pkg/commons/cluster.go`.
