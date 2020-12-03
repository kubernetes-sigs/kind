---
title: "Development"
menu:
  main:
    parent: "contributing"
    identifier: "development"
    weight: 3
toc: true
description: |-
  🚧  This page is a work-in-progress  🚧

  This page is intended to provide an introduction to developing kind. 
---

## Overview

KIND provides various utilities for development wrapped in `make`.

Most scripts require more or less only `make` + `bash` on the host, and generally
stick to POSIX utilities. Some scripts use `docker` e.g. for image building or
to use docker images containing special tools.

This guide will introduce you to important make targets and their usage.

## Building

### Building `kind`

Invoke `make build` and `bin/kind` will contain the freshly build `kind` binary
upon a successful build.

Like other targets, this target automatically manages the correct [`.go-version`][go-version] and doesn't require that you install any tooling (just `make` / `bash`). We accomplish this using a copy of [gimme] in `hack/third_party/gimme`.

### Building The Base Image

> **NOTE**: Most development should not require changes to the base image, however if your changes do, here's how to build and test it.

To build the "base image" for development use the `make quick` command in `images/base` directory: `make -C images/base quick`

By default, the base image will be tagged as `kindest/base:$(date +v%Y%m%d)-$(git describe --always --dirty)` format.
If you want to change this, you can set `TAG` environment variable.

`TAG=v0.1.0 make -C images/base quick`

For "production" base images one of the maintainers will run `make -C images/base push` which cross-compiles for all architectures and pushes to the registry.

You generally don't need to cross build during development, and currently the cross
build *must* be pushed instead of loaded locally, due to limitations in `docker buildx` (TODO: link to upstream issue).

To test out your changes take the image you built with `make quick` and use it
as the `--base-image` flag when running `kind build node-image` / building node images. You can then create a cluster with this node image (`kind create cluster --image=kindest/node:latest`)

For "Production" base image updates one of the maintainers will bump `DefaultBaseImage` in `pkg/build/nodeimage/defaults.go` to point to the newly pushed image.

### Building Node Images

// TODO: See quick-start

## Updating Generated Code

You can regenerate checked-in generated sources with `make generate`.
KIND does not use much generated code, but it does use a little.
Namely kind uses [`deepcopy-gen`] to generate `DeepCopy` methods for API types.

There is also a `make update` target meant to cover all automated code generation
+ formatting (`gofmt`).

## Testing

- Run `make test` to run all tests (unit + integration).
- Run `make unit` to run only unit tests.
- Run `make integration` to run only integration tests.

Like other targets, these targets automatically manage the correct [`.go-version`][go-version] and doesn't require that you install any tooling (just `make` / `bash`).

### E2E Testing

Coming soon ...

TLDR: `hack/ci/e2e.sh` will run e2e tests against your local Kubernetes checkout.

Depending on your changes, you may want to e2e tests.

In the future we plan to have e2e smoke tests that are cheaper / don't require
building Kubernetes.

## Linting

You can run all of our lints at once with `make verify`.

Lints include:
- checking that generated code is up to date
  - you can run just this one with `hack/make-rules/verify/generated.sh`
- [golangci-lint] with a custom config (`hack/tools/.golangci.yml`) to lint Go sources
  - you can run just this one with `make lint`
  - This linter is essentially an optimized combination of _many_ Go linters
- [shellcheck] to lint our shell scripts (invoked via docker so you don't need to install it)
  - you can run just this one with `make shellcheck`

## Documentation

// TODO

TLDR:
- basically you can run a development instance locally with `make -C site serve`.
- the site is built with hugo, the makefile only requires `docker` though
- we have a custom template under `site/layouts` & `site/assets`
- our custom theme has a few useful but simple shortcodes

## CI

// TODO

TLDR:
- prow.k8s.io
- https://git.k8s.io/test-infra/config/jobs/kubernetes-sigs/kind
- github actions for podman

[gimme]: https://github.com/travis-ci/gimme
[shellcheck]: https://shellcheck.net
[golangci-lint]: https://github.com/golangci/golangci-lint
[go-version]: https://sigs.k8s.io/kind/.go-version
