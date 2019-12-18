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

{{< codeFromFile file="static/examples/config-with-mounts.yaml" lang="yaml" >}}


### Extra Port Mappings

Extra port mappings can be used to port forward to the kind nodes. This is a 
cross-platform option to get traffic into your kind cluster. 

With docker on Linux you can simply send traffic to the node IPs from the host
without this, but to cover macOS and Windows you'll want to use these.

You may also want to see the [Ingress Guide].

{{< codeFromFile file="static/examples/config-with-port-mapping.yaml" lang="yaml" >}}


[Ingress Guide]: ./../ingress