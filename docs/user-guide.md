# User Guide

## Advanced

### Building the base and node images
#### Base image
`kind` runs a local kubernetes cluster by using Docker containers as "nodes."
`kind` uses the [`node-image`][node image] to run kubernetes artifacts, such 
as `kubeadm` or `kubelet`.
The `node-image` in turn is built off the [`base-image`][base image], which
installs all the dependencies needed for Docker and Kubernetes to run in a
container.

To build the `base-image` we use the `build` command:
```
$ kind build base-image
```

If you want to specify the path to the base image source files you can use the
`--source` flag.
If `--source` is not specified, `kind` has enbedded the contents of the in
default base image in [`pkg/build/base/sources`][pkg/build/base/sources] and
will use this to build it.

By default, the base image will be tagged as `kindest/base:latest`.
If you want to change this, you can use the `--image` flag.

```
$ kind build base-image --image base:v0.1.0
```

#### Node image
Currently, `kind` supports three different ways to build a `node-image`: via
`apt`, or if you have the [kubernetes][kubernetes] source in your host machine
(`$GOPATH/src/k8s.io/kubernetes`), by using `docker` or `bazel`.
To specified the build type use the flag `--type`.
`Kind` will default to using the build type `docker` if none is specified.

```
$ kind build node-image --type apt
```

Similarly as for the base-image command, you can specify the name and tag of
the resulting node image using the flag `--image`.

If you previously changed the name and tag of the base image, you can use here
the flag `--base-image` to specify the name and tag you used.


[node image]: ./base-image.md
[base image]: ./node-image.md
[pkg/build/base/sources]: ./../pkg/build/base/sources
[kubernetes]: https://github.com/kubernetes/kubernetes
