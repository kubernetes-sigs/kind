---
title: "Base Image"
menu:
  main:
    parent: "design"
    identifier: "base-image"
---

This page used to host a doc about the initial design, this has been found confusing
so we've updated it to clarify the current expectations. While the sources of the project
are fully open, depending on the specifics of the node image internals is not supported.

We only support that base images will create a working node image with `kind build node-image` at the kind release they were shipped with.

The contents and implemlentation of the images are subject to change at any time
to fix bugs, improve reliability, performance, or maintainability.

DO NOT DEPEND ON THE INTERNALS OF THE BASE IMAGES.

KIND provides [conformant][conformance] Kubernetes, anything else is an implementation detail.

We will not accept bugs about "breaking changes" to base images and you depend on the implementation details at your own peril.

[conformance]: https://www.cncf.io/training/certification/software-conformance/
