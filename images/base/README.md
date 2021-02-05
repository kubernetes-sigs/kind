<!--TODO(bentheelder): fill this in much more thoroughly-->
# images/base

This directory contains sources for building the `kind` base "node" image.

The image can be built with `make quick`.

## Maintenance

This image needs to do a number of unusual things to support running systemd,
nested containers, and Kubernetes. All of what we do and why we do it
is documented inline in the [Dockerfile](./Dockerfile).

If you make any changes to this image, please continue to document exactly
why we do what we do, citing upstream documentation where possible.

See also [`pkg/cluster`](./../../pkg/cluster) for logic that interacts with this image.

## Updating dependencies

If you need to change a version of containerd, crictl, or CNI, you can use the
provided script `make update-shasums` to specify the
versions and update the Dockerfile `ARG` values for you. The script will fetch
the sha256sums from GitHub releases, or will download the artifact and generate
a sha256sum.

```
$ make update-shasums

ARG CONTAINERD_AMD64_SHA256SUM=69ce75857abb424b243d3442eb9d1e96a1e853595a8562c3c03ccbdaf8fd6e59
ARG CONTAINERD_ARM64_SHA256SUM=7fc4a886466a8f0ecc80299cec03cdaca3e8b9ddf4aaa60deb9cb2b7ea0575aa
ARG CONTAINERD_PPC64LE_SHA256SUM=6536f22c38186b3826c4841d836191254ffbbab033356faebf6635778e856dd0

ARG RUNC_AMD64_SHA256SUM=64c2742b89fe0364f360b816a3c72dd8f067f49761002c5f2072c1f1e76cbad7
ARG RUNC_ARM64_SHA256SUM=91dac17a62fada7db2eb10592099f5e999e9ac1d2daf1988620656f534dee94c
ARG RUNC_PPC64LE_SHA256SUM=3ff250698360d3953a8c153e2f715d3653c58b51ecdb156f8d4cf5f17b1ece49

ARG CRICTL_AMD64_SHA256SUM=87d8ef70b61f2fe3d8b4a48f6f712fd798c6e293ed3723c1e4bbb5052098f0ae
ARG CRICTL_ARM64_SHA256SUM=ec040d14ca03e8e4e504a85dae5353e04b5d9d8aea3df68699258992c0eb8d88
ARG CRICTL_PPC64LE_SHA256SUM=72107c58960ee9405829c3366dbfcd86f163a990ea2102f3ed63a709096bc7ba

ARG CNI_PLUGINS_AMD64_SHA256SUM=58a58d389895ba9f9bbd3ef330f186c0bb7484136d0bfb9b50152eed55d9ec24
ARG CNI_PLUGINS_ARM64_SHA256SUM=49bdf1d3c852a831964aea8c9d12340b36107ee756d8328403905ff599abc6f5
ARG CNI_PLUGINS_PPC64LE_SHA256SUM=d37829b5eeca0c941b4478203c75c6cc26d9cfc1d6c8bb451c0008e0c02a025f
```

## Alternate Sources

Kind frequently picks up new releases of dependent projects including
containerd, runc, cni, and crictl. If you choose to use the provided Dockerfile
but use build arguments to specify a different base image or application version
for dependencies, be aware that you may possibly encounter bugs and undesired
behavior.

## Design

See [base-image](https://kind.sigs.k8s.io/docs/design/base-image/) for more design details.
