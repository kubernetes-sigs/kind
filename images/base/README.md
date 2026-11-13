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


## Alternate Sources

Kind frequently picks up new releases of dependent projects including
containerd, runc, cni, and crictl. If you choose to use the provided Dockerfile
but use build arguments to specify a different base image or application version
for dependencies, be aware that you may possibly encounter bugs and undesired
behavior.

## Licensing

License notices for the third-party software we build into this image are
saved under `/LICENSES` in the image.

runc is statically linked against [libpathrs], which is available under
`MPL-2.0 OR LGPL-3.0-or-later`. We elect to use libpathrs under the terms of
the **MPL-2.0**, as required by the [CNCF licensing exception] for statically
linked libpathrs. libpathrs is retrieved unmodified from upstream at build time,
and the image includes a notice at `/LICENSES/cyphar/libpathrs/SOURCE` pointing
to the exact source code used.

[libpathrs]: https://github.com/cyphar/libpathrs
[CNCF licensing exception]: https://github.com/cncf/foundation/blob/main/license-exceptions/CNCF-licensing-exceptions.csv

## Design

See [base-image](https://kind.sigs.k8s.io/docs/design/base-image/) for more design details.
