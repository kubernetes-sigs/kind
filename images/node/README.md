<!--TODO(bentheelder): fill this in much more thoroughly-->
# images/node

This directory contains sources for building the `kind` "node" image.

The image can be built with `kind build image`.

## Maintenance

This image needs to do a number of unusual things to support running systemd, 
nested containers, and Kubernetes. All of what we do and why we do it 
is documented inline in the [Dockerfile](./Dockerfile).

If you make any changes to this image, please continue to document exactly 
why we do what we do, citing upstream documentation where possible.

See also [`pkg/cluster`](./../../pkg/cluster) for logic that interacts directly with this image.
