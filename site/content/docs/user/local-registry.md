---
title: "Local Registry"
menu:
  main:
    parent: "user"
    identifier: "user-local-registry"
    weight: 3
description: |-
  This guide covers how to configure KIND with a local container image registry.

  In the future this will be replaced by [a built-in feature](https://github.com/kubernetes-sigs/kind/issues/1213), and this guide will
  cover usage instead.
---
## Create A Cluster And Registry

The following shell script will create a local docker registry and a kind cluster
with it enabled.

{{< codeFromFile file="static/examples/kind-with-registry.sh" >}}

## Using The Registry

The registry can be used like this.

1. First we'll pull an image `docker pull gcr.io/google-samples/hello-app:1.0`
2. Then we'll tag the image to use the local registry `docker tag gcr.io/google-samples/hello-app:1.0 localhost:5001/hello-app:1.0`
3. Then we'll push it to the registry `docker push localhost:5001/hello-app:1.0`
4. And now we can use the image `kubectl create deployment hello-server --image=localhost:5001/hello-app:1.0`

If you build your own image and tag it like `localhost:5001/image:foo` and then use
it in kubernetes as `localhost:5001/image:foo`. And use it from inside of your cluster application as `kind-registry:5000`.
