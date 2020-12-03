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

// TODO

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
