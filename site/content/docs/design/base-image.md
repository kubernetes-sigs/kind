---
title: "Base Image"
menu:
  main:
    parent: "design"
    identifier: "base-image"
---

> **NOTE**: This may not completely cover the current implementation.

The ["base" image][base image] is a small-ish Docker image for running
nested containers, systemd, and kubernetes components.

To do this we need to set up an environment that will meet the CRI 
(currently containerd and runc) and systemd's particular needs. Documentation for each
step we take is inline to the image's [Dockerfile][dockerfile],
but essentially:

- we preinstall tools / packages expected by systemd / containerd / Kubernetes other
than Kubernetes itself

- we install a custom entrypoint that allows us to perform some actions before
the container truly boots

- we set up a systemd service to forward journal logs to the container tty

- we do a few tricks to minimize unnecessary services and inform systemd that it
is in docker (see the [Dockerfile][dockerfile])

This image is based on the `debian-slim` image which starts relatively small for
a Kubernetes node image, has near exclusively packages we need, and has
relatively up to date packages.
We strive to minimize the image size where possible.

[base image]: https://sigs.k8s.io/kind/images/base
[dockerfile]: https://sigs.k8s.io/kind/images/base/Dockerfile
