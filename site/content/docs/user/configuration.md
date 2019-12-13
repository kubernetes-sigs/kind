---
title: "Configuration"
menu:
  main:
    parent: "user"
    identifier: "user-configuration"
    weight: 3
---

<h1>⚠️ This page is a stub! We're working on it! ⚠️ </h1>

# Configuration

This guide covers how to configure KIND cluster creation.

We know this is currently a bit lacking right now and will expand it over time - PRs welcome!

<!--TODO: TOC?-->

## Getting Started

To configure kind cluster creation, you will need to create a [YAML] config file.
This file follows Kubernetes conventions for versioning etc. <!--todo links for this-->

A minimal valid config is:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
```

This config merely specifies that we are configuration a KIND cluster (`kind: Cluster`)
and that the version of KIND's config we are using is `v1alpha4` (`apiVersion: kind.x-k8s.io/v1alpha4`).

Any given version of kind may support different versions which will have different
options and behavior. This is why we must always specify the version.

This mechanism is inspired by Kubernetes resources and component config.

To use this config, place the contents in a file `config.yaml` and then run
`kind create cluster --config=config.yaml` from the same directory.

You can also include a full file path like `kind create cluster --config=/foo/bar/config.yaml`.

## Options

The following high level options are available.

NOTE: not all options are documented yet!  We will fix this with time, PRs welcome!

### Extra Mounts

Extra mounts can be used to pass through storage on the host to a kind node
for persisting data, mounting through code etc.

{{< codefromfile file="examples/config-with-mounts.yaml" lang="yaml" >}}

