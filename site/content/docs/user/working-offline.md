---
title: "Working Offline"
menu:
  main:
    parent: "user"
    identifier: "working-offline"
    weight: 3
description: |-
  This guide covers how to work with KIND in an offline / airgapped environment.

  You should first [install kind][installation documentation] before continuing.

  [installation documentation]: https://kind.sigs.k8s.io/docs/user/quick-start#installation
---
## Using a pre-built [node image][node image]

KIND provides some pre-built images,
these images contain everything necessary to create a cluster and can be used in an offline environment.

You can find available image tags on the [releases page][releases page].
Please include the `@sha256:` [image digest][image digest] from the image in the release notes.

You can pull it when you have network access,
or pull it on another machine and then transfer it to the target machine.

```
➜  ~ docker pull kindest/node:v1.17.0@sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62
sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62: Pulling from kindest/node
cc5a81c29aab: Pull complete 
81c62728355f: Pull complete 
ed9cffdd962a: Pull complete 
6a46f000fce2: Pull complete 
6bd890da28be: Pull complete 
0d88bd219ffe: Pull complete 
af5240f230f0: Pull complete 
Digest: sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62
Status: Downloaded newer image for kindest/node@sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62
docker.io/kindest/node:v1.17.0@sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62
```

You can [save node image][docker save] to a tarball.

```
➜  ~ docker save -o kind.v1.17.0.tar kindest/node:v1.17.0@sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62
# or
➜  ~ docker save kindest/node:v1.17.0@sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62 | gzip > kind.v1.17.0.tar.gz
```

When you transport image tarball to the machine,
you can load the node image by [`docker load`][docker load] command.

```
➜  ~ docker load -i kind.v1.17.0.tar
Loaded image ID: sha256:ec6ab22d89efc045f4da4fc862f6a13c64c0670fa7656fbecdec5307380f9cb0
# or
➜  ~ docker load -i kind.v1.17.0.tar.gz
Loaded image ID: sha256:ec6ab22d89efc045f4da4fc862f6a13c64c0670fa7656fbecdec5307380f9cb0
```

And [create a tag][docker tag] for it.

```
➜  ~ docker image tag kindest/node:v1.17.0@sha256:9512edae126da271b66b990b6fff768fbb7cd786c7d39e86bdf55906352fdf62 kindest/node:v1.17.0
➜  ~ docker image ls kindest/node
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
kindest/node        v1.17.0             ec6ab22d89ef        3 weeks ago         1.23GB
```

Finally, you can create a cluster by specifying the `--image` flag.

```
➜  ~ kind create cluster --image kindest/node:v1.17.0
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.17.0) 🖼
 ✓ Preparing nodes 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

Have a question, bug, or feature request? Let us know! https://kind.sigs.k8s.io/#community 🙂
```

## Building the [node image][node image]

In addition to using pre-built node image, 
KIND also provides the ability to build [node image][node image] from Kubernetes source code.

Please note that during the image building process, you need to download many dependencies.
It is recommended that you build at least once online to ensure that these dependencies are downloaded to your local.
See [building the node image][building the node image] for more detail.

The node-image in turn is built off the [base image][base image].

### Prepare Kubernetes source code

You can clone Kubernetes source code.

```sh
➜  ~ mkdir -p $GOPATH/src/k8s.io
➜  ~ cd $GOPATH/src/k8s.io
➜  ~ git clone https://github.com/kubernetes/kubernetes
```

### Building image

```sh
➜  ~ kind build node-image --image kindest/node:main --kube-root $GOPATH/src/k8s.io/kubernetes 
Starting to build Kubernetes
...
Image build completed.
```

When the image build is complete, you can create a cluster by passing the `--image` flag.

```sh
➜  ~ kind create cluster --image kindest/node:main
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:main) 🖼
 ✓ Preparing nodes 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

Have a question, bug, or feature request? Let us know! https://kind.sigs.k8s.io/#community 🙂
```

## HA cluster

If you want to create a control-plane HA cluster
then you need to create a config file and use this file to start the cluster.

```sh
➜  ~ cat << EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
# 3 control plane node and 1 workers
nodes:
- role: control-plane
- role: control-plane
- role: control-plane
- role: worker
EOF
```

Note that in an offline environment, in addition to preparing the node image,
you also need to prepare HAProxy image in advance.

You can find the specific tag currently in use at [loadbalancer source code][loadbalancer source code].







[installation documentation]: https://kind.sigs.k8s.io/docs/user/quick-start#installation
[node image]: https://kind.sigs.k8s.io/docs/design/node-image
[releases page]: https://github.com/kubernetes-sigs/kind/releases
[image digest]: https://docs.docker.com/engine/reference/commandline/pull/#pull-an-image-by-digest-immutable-identifier
[docker save]: https://docs.docker.com/engine/reference/commandline/save/
[docker load]: https://docs.docker.com/engine/reference/commandline/load/
[docker tag]: https://docs.docker.com/engine/reference/commandline/tag/
[base image]: https://kind.sigs.k8s.io/docs/design/base-image/
[building the node image]: https://kind.sigs.k8s.io/docs/user/quick-start/#building-images
[loadbalancer source code]: https://github.com/kubernetes-sigs/kind/blob/main/pkg/cluster/internal/loadbalancer/const.go#L20
