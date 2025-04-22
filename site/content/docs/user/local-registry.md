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
it in kubernetes as `localhost:5001/image:foo`. 

If for some reason you have code running *inside* of a pod within the cluster that
needs to use this registry directly (e.g. to build and push an image) then that
code will need to use the `kind-registry:5000` HTTP endpoint directly, as the
code running inside your pod will not see the containerd config.

Pod manifests / pod specs / pod YAML should use `localhost:5001`,
which will be rerouted to match the same name as the host via the containerd config.

<!--TODO: consider a shared guide for this which we can use across the docs-->
> **NOTE**: A bit about "localhost" and containers ...
> "localhost" resolves to a loopback IP, which are network-namespace local.
> Network-namespace local means that the `127.0.0.1` / `localhost` / `::1`
> inside your container is NOT the same as the one on your host
> (unless using `hostNetwork: true` / `--net=host`).
>
> KIND nodes are *not* `--net=host` and most pods are not, so really these
> "`localhost`s" are local to your pod container, to your kind node, and to your
> host machine in most cases.
>
> However we can tell containerd that `localhost:5001` should route to the
> registry container such that the same name works on both the host and your machine.
>
> We do this so you can conveniently push and pull from a "local"
> address on your host and in your pod YAML.
