# haproxy

This image is used internally by kind to implement kubeadm's "HA" mode,
specifically to load balance the API server.

We cannot merely use the upstream haproxy image as haproxy will exit without
a minimal config, so we introduce one that will list on the intended port and
hot reload it at runtime with the actual desired config.

## Building

You can `make quick` in this directory to build a test image.

To push an actual image use `make push`.
