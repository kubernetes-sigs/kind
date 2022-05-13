# local-path-provisioner

This image packages https://github.com/rancher/local-path-provisioner to meet
our requirements.

- Not based on alpine (see: https://github.com/kubernetes/kubernetes/issues/109406#issuecomment-1103479928)
- Control over building with current patched go version

## Building

You can `make quick` in this directory to build a test image.

To push an actual image use `make push`.
