## images/node

See: [`pkg/build/node/node.go`][pkg/build/node/node.go], this
image is built programmatically with docker run / exec / commit for performance
reasons with large artifacts.

Roughly this image is [the base image](./../base), with the addition of:
 - installing the Kubernetes packages / binaries
 - placing the Kubernetes docker images in `/kind/images/*.tar`
 - placing a file in `/kind/version` containing the Kubernetes semver

See [`node-image`][node-image.md] for more design details.

[pkg/build/node/node.go]: ./../../pkg/build/node/node.go
[node-image.md]: https://kind.sigs.k8s.io/docs/design/node-image
