---
title: "Working Offline"
menu:
  main:
    parent: "user"
    identifier: "working-offline"
    weight: 3
---
# Working Offline

Some users may work in an offline environment,
let's talk about how to create a cluster using kind in this environment.

You can first prepare the necessary environment according to the [installation documentation][installation documentation].


## Using a pre-built [node image][node image]

Kind provided some pre-built images,
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


[installation documentation]: https://kind.sigs.k8s.io/docs/user/quick-start#installation
[node image]: https://kind.sigs.k8s.io/docs/design/node-image
[releases page]: https://github.com/kubernetes-sigs/kind/releases
[image digest]: https://docs.docker.com/engine/reference/commandline/pull/#pull-an-image-by-digest-immutable-identifier
[docker save]: https://docs.docker.com/engine/reference/commandline/save/
[docker load]: https://docs.docker.com/engine/reference/commandline/load/
[docker tag]: https://docs.docker.com/engine/reference/commandline/tag/
