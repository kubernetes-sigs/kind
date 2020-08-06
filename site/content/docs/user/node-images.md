---
title: "Node Images"
menu:
  main:
    parent: "user"
    identifier: "node-images"
    weight: 3
---
# Node Images

kind runs a local Kubernetes cluster by using Docker containers as "nodes".
kind uses the [`node-image`][node image] to run Kubernetes artifacts, such
as `kubeadm` or `kubelet`.
The `node-image` in turn is built off the [`base-image`][base image], which
installs all the dependencies needed for Docker and Kubernetes to run in a
container.

See [building the base image](#building-the-base-image) for more advanced information.

## Building Images

> Note: If you're using Docker Desktop, be sure to read [Settings for Docker Desktop](#settings-for-docker-desktop) first.


Currently, kind supports two different ways to build a `node-image`
if you have the [Kubernetes][kubernetes] source in your host machine
(`$GOPATH/src/k8s.io/kubernetes`), by using `docker` or `bazel`.
To specify the build type use the flag `--type`.
Note however that using `--type=bazel` on Windows or MacOS will not work
currently due to Kubelet using [CGO] which requires GCC/glibc for linux.
A workaround may be enabled in the future.

kind will default to using the build type `docker` if none is specified.

```
kind build node-image --type bazel
```

Similarly as for the base-image command, you can specify the name and tag of
the resulting node image using the flag `--image`.

If you previously changed the name and tag of the base image, you can use here
the flag `--base-image` to specify the name and tag you used.

### Settings for Docker Desktop

If you are building Kubernetes (for example - `kind build node-image`) on MacOS or Windows then you need a minimum of 6GB of RAM
dedicated to the virtual machine (VM) running the Docker engine. 8GB is recommended.

To change the resource limits for the Docker on Mac, you'll need to open the
**Preferences** menu.
<img src="/docs/user/images/docker-pref-1.png"/>

Now, go to the **Advanced** settings page, and change the
settings there, see [changing Docker's resource limits][Docker resource lims].
<img src="/docs/user/images/docker-pref-2.png" alt="Setting 8Gb of memory in Docker for Mac" />


To change the resource limits for the Docker on Windows, you'll need to right-click the Moby
icon on the taskbar, and choose "Settings". If you see "Switch to Linux Containers", then you'll need
to do that first before opening "Settings"

<img src="/docs/user/images/docker-pref-1-win.png"/>

Now, go to the **Advanced** settings page, and change the
settings there, see [changing Docker's resource limits][Docker resource lims].

<img src="/docs/user/images/docker-pref-build-win.png" alt="Setting 8Gb of memory in Docker for Windows" />


You may also try removing any unused data left by the Docker engine - e.g.,
`docker system prune`.

## Customizing node images

kind images are heavily optimized to be as light as possible, this has the downside that some common programs are missing, such as an editor, ping, ... There is also users that need more specific tools for their clusters or want to embed additional container images to avoid loading once the cluster is up.

It is easy to create a new node image based on a existing one with the additional software, just leveraging the containers capabilities, per example, creating a new Dockerfile that uses the official image as base and adds an step to install the additional software needed. For example:

```
FROM kindest/node:v1.18.2
RUN apt-get update && apt-get install -y \
    iputils-ping \
    tcpdump \
    vim \
 && rm -rf /var/lib/apt/lists/*

COPY my-docker-image.tar /kind
```

Build the new image and tag it:

```sh
docker build -t mykindnode:mytag .
```

And use it to create the cluster:

```sh
kind create cluster --image mykindnode:mytag
Creating cluster "kind" ...
 ✓ Ensuring node image (mykindnode:mytag) 🖼
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-kind"
You can now use your cluster with:
kubectl cluster-info --context kind-kind
Have a nice day! 👋
```

Once the cluster is ready you can use the new software that was previously installed
in the node images:

```sh
docker exec -it kind-control-plane ping -c 3 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=115 time=23.3 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=115 time=23.1 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=115 time=22.5 ms
```

## Building The Base Image

To build the `base-image` we use the `make quick` command in `images/base` directory:
```
make quick
```

By default, the base image will be tagged as `kindest/base:$(date +v%Y%m%d)-$(git describe --always --dirty)` format.
If you want to change this, you can set `TAG` environment variable.

```
TAG=v0.1.0 make quick
```

[node image]: /docs/design/node-image
[base image]: /docs/design/base-image
[Docker resource lims]: https://docs.docker.com/docker-for-mac/#advanced
[kubernetes]: https://github.com/kubernetes/kubernetes
[CGO]: https://golang.org/cmd/cgo/