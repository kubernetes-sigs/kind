---
title: "Base Image"
menu:
  main:
    parent: "design"
    identifier: "base-image"
---
# The Base Image

**NOTE**: This may not completely cover the current implementation.

The ["base" image][base image] is a small-ish Docker image for running
nested containers, systemd, and kubernetes components.

To do this we need to set up an environment that will meet the CRI 
(currently just docker) and systemd's particular needs. Documentation for each
step we take is inline to the image's [Dockerfile][dockerfile]),
but essentially:

- we preinstall tools / packages expected by systemd / Docker / Kubernetes other
than Kubernetes itself

- we install a custom entrypoint that allows us to perform some actions before
the container truly boots

- we set up a systemd service to forward journal logs to the container tty

- we do a few tricks to minimize unnecessary services and inform systemd that it
is in docker (see the [Dockerfile][dockerfile])

This image is based on `ubuntu:18.04` image
due to high availability of tooling.
We strive to minimize the image size where possible.

[base image]: https://sigs.k8s.io/kind/images/base
[dockerfile]: https://sigs.k8s.io/kind/images/base/Dockerfile
