## images/node

See: [./../../pkg/build/node_image.go](./../../pkg/build/node_image.go), this
image is built programmatically with docker run / exec / commit for performance
reasons with large artifacts.

Roughly this image is [the base image](./../base), with the addition of:
 - installing the Kubernetes packages / binaries
 - placing the Kubernetes docker images in /kind/images/*.tar
 - placing a file in /kind/version containing the Kubernetes semver
