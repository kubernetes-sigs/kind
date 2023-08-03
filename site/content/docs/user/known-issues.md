---
title: "Known Issues"
menu:
  main:
    parent: "user"
    identifier: "known-issues"
    weight: 2
description: |-
  Having problems with kind? This guide covers some known problems and solutions / workarounds.

  It may additionally be helpful to:

  - check our [issue tracker]
  - [file an issue][file an issue] (if there isn't one already)
  - reach out and ask for help in [#kind] on the [kubernetes slack]

  [issue tracker]: https://github.com/kubernetes-sigs/kind/issues
  [file an issue]: https://github.com/kubernetes-sigs/kind/issues/new
  [#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
  [kubernetes slack]: https://slack.k8s.io/
---



## Contents
* [Troubleshooting Kind](#troubleshooting-kind)
* [Kubectl Version Skew](#kubectl-version-skew) (Kubernetes limits supported version skew)
* [Older Docker Installations](#older-docker-installations) (untested, known to have bugs)
* [Docker Installed With Snap](#docker-installed-with-snap) (snap filesystem restrictions problematic)
* [Failure to Build Node Image](#failure-to-build-node-image) (usually need to increase resources)
* [Failing to Properly Start Cluster](#failing-to-properly-start-cluster) (various causes)
* [Failure to Create Cluster with Cgroups v2](#failure-to-create-cluster-with-cgroups-v2) (only supported for Kubernetes >= 1.19)
* [Pod Errors Due to "too many open files"](#pod-errors-due-to-too-many-open-files) (likely [inotify] limits which are not namespaced)
* [Docker Permission Denied](#docker-permission-denied) (ensure you have permission to use docker)
* [Windows Containers](#windows-containers) (unsupported / infeasible)
* [Non-AMD64 Architectures](#nonamd64-architectures) (images not pre-built yet)
* [Unable to Pull Images](#unable-to-pull-images) (various)
* [Chrome OS](#chrome-os) (needs KubeletInUserNamespace)
* [AppArmor](#apparmor) (may break things, consider disabling)
* [IPv6 Port Forwarding](#ipv6-port-forwarding) (docker doesn't seem to implement this correctly)
* [Couldn't find an alternative telinit implementation to spawn](#docker-init-daemon-config)
* [Fedora](#fedora) (various)
* [Failed to get rootfs info](#failed-to-get-rootfs-info--stat-failed-on-dev)
* [Failure to Create Cluster with Docker Desktop as Container Runtime](#failure-to-create-cluster-with-docker-desktop-as-container-runtime)
* [Docker Desktop for macOS and Windows](#docker-desktop-for-macos-and-windows)

## Troubleshooting Kind

If the cluster fails to create, try again with the `--retain` option (preserving the failed container),
then run `kind export logs` to export the logs from the container to a temporary directory on the host.

## Kubectl Version Skew

You may have problems interacting with your kind cluster if your client(s) are
skewed too far from the kind node version. Kubernetes [only supports limited skew][version skew]
between clients and the API server.

This is a issue that frequently occurs when running `kind` alongside Docker For Mac.

This problem is related to a bug in [docker on macOS][for-mac#3663]

If you see something like the following error message:
```bash
$ kubectl edit deploy -n kube-system kubernetes-dashboard
error: SchemaError(io.k8s.api.autoscaling.v2beta1.ExternalMetricStatus): invalid object doesn't have additional properties
```

You can check your client and server versions by running:
{{< codeFromInline lang="bash" >}}
kubectl version
{{< /codeFromInline >}}

If there is a mismatch between the server and client versions, you should install a newer client version.

If you are using Mac, you can install kubectl via homebrew by running:
{{< codeFromInline lang="bash" >}}
brew install kubernetes-cli
{{< /codeFromInline >}}

And overwrite the symlinks created by Docker For Mac by running:
{{< codeFromInline lang="bash" >}}
brew link --overwrite kubernetes-cli
{{< /codeFromInline >}}

[for-mac#3663]: https://github.com/docker/for-mac/issues/3663

## Older Docker Installations

> **NOTE**: This only applies to kind version v0.15.0 and back: Kubernetes
> before 1.15 will not be supported in KIND v0.16.0 and versions below 1.13
> were no longer supported in kind v0.9.0.

`kind` is known to have issues with Kubernetes 1.13 or lower when using Docker versions:

- `1.13.1` (released January 2017)
- `17.05.0-ce` (released May 2017)

And possibly other old versions of Docker.

With these versions you must use Kubernetes >= 1.14, or more ideally upgrade Docker instead.

kind is tested with a recent stable `docker-ce` release.

## Docker Installed with Snap

If you installed Docker with [snap], it is likely that `docker` commands do not
have access to `$TMPDIR`. This may break some kind commands which depend
on using temp directories (`kind build ...`).

Currently a workaround for this is setting the `TEMPDIR` environment variable to
a directory snap does have access to when working with kind.
This can for example be some directory under `$HOME`.


## Failure to build node image
Building kind's node image may fail due to running out of memory on Docker for Mac or Docker for Windows.
See [kind#229][kind#229].

If you see something like this:

```
    cmd/kube-scheduler
    cmd/kube-proxy
/usr/local/go/pkg/tool/linux_amd64/link: signal: killed
!!! [0116 08:30:53] Call tree:
!!! [0116 08:30:53]  1: /go/src/k8s.io/kubernetes/hack/lib/golang.sh:614 kube::golang::build_some_binaries(...)
!!! [0116 08:30:53]  2: /go/src/k8s.io/kubernetes/hack/lib/golang.sh:758 kube::golang::build_binaries_for_platform(...)
!!! [0116 08:30:53]  3: hack/make-rules/build.sh:27 kube::golang::build_binaries(...)
!!! [0116 08:30:53] Call tree:
!!! [0116 08:30:53]  1: hack/make-rules/build.sh:27 kube::golang::build_binaries(...)
!!! [0116 08:30:53] Call tree:
!!! [0116 08:30:53]  1: hack/make-rules/build.sh:27 kube::golang::build_binaries(...)
make: *** [all] Error 1
Makefile:92: recipe for target 'all' failed
!!! [0116 08:30:54] Call tree:
!!! [0116 08:30:54]  1: build/../build/common.sh:518 kube::build::run_build_command_ex(...)
!!! [0116 08:30:54]  2: build/release-images.sh:38 kube::build::run_build_command(...)
make: *** [quick-release-images] Error 1
ERRO[08:30:54] Failed to build Kubernetes: failed to build images: exit status 2
Error: error building node image: failed to build kubernetes: failed to build images: exit status 2
Usage:
  kind build node-image [flags]

Flags:
      --base-image string   name:tag of the base image to use for the build (default "kindest/base:v20181203-d055041")
  -h, --help                help for node-image
      --image string        name:tag of the resulting image to be built (default "kindest/node:latest")
      --kube-root string    Path to the Kubernetes source directory (if empty, the path is autodetected)
      --type string         build type, default is docker (default "docker")

Global Flags:
      --loglevel string   logrus log level [panic, fatal, error, warning, info, debug] (default "warning")

error building node image: failed to build kubernetes: failed to build images: exit status 2
```

Then you may try increasing the resource limits for the Docker engine on Mac or Windows.

It is recommended that you allocate at least 8GB of RAM to build Kubernetes.

Open the **Preferences** menu.

<img src="/docs/user/images/docker-pref-1.png"/>

Go to the **Advanced** settings page, and change the settings there, see
[changing Docker's resource limits][Docker resource lims].

<img width="400px" src="/docs/user/images/docker-pref-build.png" alt="Setting 8Gb of memory in Docker for Mac" />

<img width="400px" src="/docs/user/images/docker-pref-build-win.png" alt="Setting 8Gb of memory in Docker for Windows" />

## Failing to properly start cluster
This issue is similar to a
[failure while building the node image](#failure-to-build-node-image).
If the cluster creation process was successful but you are unable to see any
Kubernetes resources running, for example:

```
$ docker ps
CONTAINER ID        IMAGE                  COMMAND                  CREATED              STATUS              PORTS                      NAMES
c0261f7512fd        kindest/node:v1.12.2   "/usr/local/bin/entr…"   About a minute ago   Up About a minute   0.0.0.0:64907->64907/tcp   kind-1-control-plane
$ docker exec -it c0261f7512fd /bin/sh
# docker ps -a
CONTAINER ID        IMAGE               COMMAND             CREATED             STATUS              PORTS               NAMES
#
```

or `kubectl` being unable to connect to the cluster,
```
$ kind export kubeconfig
$ kubectl cluster-info

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
Unable to connect to the server: EOF
```

Then as in [kind#156][kind#156], you may solve this issue by claiming back some
space on your machine by removing unused data or images left by the Docker
engine by running:
{{< codeFromInline lang="bash" >}}
docker system prune
{{< /codeFromInline >}}

And / or:
{{< codeFromInline lang="bash" >}}
docker image prune
{{< /codeFromInline >}}

You can verify the issue by exporting the logs (`kind export logs`) and looking
at the kubelet logs, which may have something like the following:
```
Dec 07 00:37:53 kind-1-control-plane kubelet[688]: I1207 00:37:53.229561     688 eviction_manager.go:340] eviction manager: must evict pod(s) to reclaim ephemeral-storage
Dec 07 00:37:53 kind-1-control-plane kubelet[688]: E1207 00:37:53.229638     688 eviction_manager.go:351] eviction manager: eviction thresholds have been met, but no pods are active to evict
```
## Failure to Create Cluster with Cgroups v2
Support for Cgroups v2 was introduced in Kubernetes 1.19 (see [release notes](https://v1-19.docs.kubernetes.io/docs/setup/release/notes/)). Accordingly, only Kubernetes versions >= 1.19 are supported on hosts using Cgroups v2.

You can verify that you are hitting this issue by exporting the logs (`kind export logs`) and looking
at the kubelet logs, which may have something like the following:
```
Feb 23 08:39:04 kind-control-plane kubelet[3031]: W0223 08:39:04.644052    3031 server.go:616] failed to get the kubelet's cgroup: mountpoint for cpu not found.  Kubelet system container metrics may be missing.
Feb 23 08:39:04 kind-control-plane kubelet[3031]: F0223 08:39:04.644296    3031 server.go:274] failed to run Kubelet: mountpoint for  not found
```
## Pod errors due to "too many open files"

This may be caused by running out of [inotify](https://linux.die.net/man/7/inotify) resources. Resource limits are defined by `fs.inotify.max_user_watches` and `fs.inotify.max_user_instances` system variables. For example, in Ubuntu these default to 8192 and 128 respectively, which is not enough to create a cluster with many nodes.

To increase these limits temporarily run the following commands on the host:
{{< codeFromInline lang="bash" >}}
sudo sysctl fs.inotify.max_user_watches=524288
sudo sysctl fs.inotify.max_user_instances=512
{{< /codeFromInline >}}

To make the changes persistent, edit the file `/etc/sysctl.conf` and add these lines:
{{< codeFromInline lang="bash" >}}
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 512
{{< /codeFromInline >}}

## Docker permission denied

When using `kind`, we assume that the user you are executing kind as has permission to use docker.
If you initially ran Docker CLI commands using `sudo`, you may see the following error, which indicates that your `~/.docker/` directory was created with incorrect permissions due to the `sudo` commands.

```
WARNING: Error loading config file: /home/user/.docker/config.json
open /home/user/.docker/config.json: permission denied
```

To fix this problem, either follow the docker's docs [manage docker as a non root user][manage docker as a non root user],
or try to use `sudo` before your commands (if you get `command not found` please check [this comment about sudo with kind][sudo with kind]).


## Docker init daemon config

Please make sure that when you use `kind`, you can't have `"init": true` in your `/etc/docker/daemon.json` because that will
cause `/sbin/init` to show the following cryptic message *Couldn't find an alternative telinit implementation to spawn*.
This has to to with `/sbin/init` not running as process id 1.

## Windows Containers

[Docker Desktop for Windows][docker desktop for windows] supports running both Linux (the default) and Windows Docker containers.

`kind` for Windows requires Linux containers. To switch between Linux and Windows containers see [this page][switch between windows and linux containers].

Windows containers are not like Linux containers and do not support running docker in docker and therefore cannot support kind.

## Non-AMD64 Architectures

KIND does not currently ship pre-built images for non-amd64 architectures.
In the future we may, but currently demand has been low and the cost to build
has been high.

To use kind on other architectures, you need to first build a base image
and then build a node image.

Run `images/base/build.sh` and then taking note of the built image name use `kind build node-image --base-image=kindest/base:tag-i-built`.

There are more details about how to do this in the [Quick Start] guide.

## Unable to pull images

When using named KIND instances you may sometimes see your images failing to pull correctly on pods. This will usually manifest itself with the following output when doing a `kubectl describe pod my-pod`

```
Failed to pull image "docker.io/my-custom-image:tag": rpc error: code = Unknown desc = failed to resolve image "docker.io/library/my-custom-image:tag": no available registry endpoint: pull access denied, repository does not exist or may require authorization: server message: insufficient_scope: authorization failed
```

If this image has been loaded onto your kind cluster using the command `kind load docker-image my-custom-image` then you have likely not provided the name parameter.

Re-run the command this time adding the `--name my-cluster-name` param:

`kind load docker-image my-custom-image --name my-cluster-name`

## Chrome OS

To run Kubernetes inside Chrome OS the LXC container must allow nesting. In Crosh session (ctrl+alt+t):

```
crosh> vmc launch termina
(termina) chronos@localhost ~ $ lxc config set penguin security.nesting true
(termina) chronos@localhost ~ $ lxc restart penguin
```

Then KIND cluster must use KubeletInUserNamespace feature gate (available since Kubernetes 1.22):

```
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
featureGates:
  KubeletInUserNamespace: true
```

## AppArmor

If your host has [AppArmor] enabled you may run into [moby/moby/issues/7512](https://github.com/moby/moby/issues/7512#issuecomment-51845976).

You will likely need to disable apparmor on your host or at least any profile(s)
related to applications you are trying to run in KIND.

See Previous Discussion: [kind#1179]

## IPv6 Port Forwarding

Docker assumes that all the IPv6 addresses should be reachable, hence doesn't implement
port mapping using NAT [moby#17666].

You will likely need to use Kubernetes services like NodePort or LoadBalancer to access
your workloads inside the cluster via the nodes IPv6 addresses.

See Previous Discussion: [kind#1326]

## Failed to get rootfs info / "stat failed on /dev/..."

On some systems, creating a cluster times out with these errors in kubelet.log (device varies):

```
stat failed on /dev/nvme0n1p3 with error: no such file or directory
"Failed to start ContainerManager" err="failed to get rootfs info: failed to get device for dir \"/var/lib/kubelet\": could not find device with major: 0, minor: 40 in cached partitions map"
```

Kubernetes needs access to storage device nodes in order to do some stuff, e.g. tracking free disk space. Therefore, Kind needs to mount the necessary device nodes from the host into the control-plane container — however, it cannot always determine which device Kubernetes requires, since this varies with the host OS and filesystem. For example, the error above occurred with a BTRFS filesystem on Fedora Desktop 35.

This can be worked around by including the necessary device as an extra mount in the cluster configuration file.

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /dev/nvme0n1p3
      containerPath: /dev/nvme0n1p3
      propagation: HostToContainer
```

To identify the device that must be listed, two variations have been observed.
* The device reported in the error message is a symlink (e.g. `/dev/mapper/luks-903aad3d-...`) — in this case, the config file should refer to the target of that symlink (e.g. `/dev/dm-0`).
* The device reported in the error message is a regular block device (e.g. `/dev/nvme0n1p3`) — in this case, use the device reported.

See Previous Discussion: [kind#2411]

## Fedora

### Firewalld

On Fedora 32 [firewalld] moved to nftables backend by default.
This seems to be incompatible with Docker, leading to KIND cluster nodes not
being able to reach each other.

You can work around this by changing the `FirewallBackend` in the `/etc/firewalld/firewalld.conf ` file from `nftables` to `iptables` and restarting firewalld.

```console
sed -i /etc/firewalld/firewalld.conf 's/FirewallBackend=.*/FirewallBackend=iptables/'
systemctl restart firewalld
```

See [#1547 (comment)](https://github.com/kubernetes-sigs/kind/issues/1547#issuecomment-623756313)
and [Docker and Fedora 32 article](https://fedoramagazine.org/docker-and-fedora-32/)

### SELinux

On Fedora 33 an update to the SELinux policy causes `kind create cluster` to fail with an error like

```sh
docker: Error response from daemon: open /dev/dma_heap: permission denied.
```

Although the policy has been fixed in Fedora 34, the fix has not been backported to Fedora 33 as of June 28, 2021. Putting SELinux in permissive mode (`setenforce 0`) is one known workaround. This disables SELinux until the next boot. For more details, see [kind#2296].

## Failure to Create Cluster with Docker Desktop as Container Runtime

Docker Desktop 4.3.0+ uses Cgroups v2 (see [release notes](https://docs.docker.com/release-notes/)) and can only work with Kubernetes versions >= 1.19 (see [failure to create cluster with Cgroups v2](https://kind.sigs.k8s.io/docs/user/known-issues/#failure-to-create-cluster-with-cgroups-v2)).

There are a couple options for getting around this limitation with Docker
Desktop. You may downgrade your Docker Desktop to version 4.2.0. This was the
last release to default to cgroupv1.

You may also be able to make a configuration change to your Docker Desktop
settings. Starting with Docker Desktop 4.4.2 a deprecated option has been added
to support those requiring cgroupv1. As noted in the [4.4.2 release
notes](https://docs.docker.com/desktop/mac/release-notes/#docker-desktop-442),
the setting `deprecatedCgroupv1` can be set to `true` in `settings.json`. After
restarting the Docker Engine, the VM used by Docker Desktop will use cgroupv1.

[issue tracker]: https://github.com/kubernetes-sigs/kind/issues
[file an issue]: https://github.com/kubernetes-sigs/kind/issues/new
[#kind]: https://kubernetes.slack.com/messages/CEKK1KTN2/
[kubernetes slack]: https://slack.k8s.io/
[kind#136]: https://github.com/kubernetes-sigs/kind/issues/136
[kind#136-docker]: https://github.com/kubernetes-sigs/kind/issues/136#issuecomment-457015838
[kind#156]: https://github.com/kubernetes-sigs/kind/issues/156
[kind#182]: https://github.com/kubernetes-sigs/kind/issues/182
[kind#200]: https://github.com/kubernetes-sigs/kind/issues/200
[kind#229]: https://github.com/kubernetes-sigs/kind/issues/229
[kind#270]: https://github.com/kubernetes-sigs/kind/issues/270
[kind#1179]: https://github.com/kubernetes-sigs/kind/issues/1179
[kind#1326]: https://github.com/kubernetes-sigs/kind/issues/1326
[kind#2296]: https://github.com/kubernetes-sigs/kind/issues/2296
[kind#2411]: https://github.com/kubernetes-sigs/kind/issues/2411
[moby#9939]: https://github.com/moby/moby/issues/9939
[moby#17666]: https://github.com/moby/moby/issues/17666
[Docker resource lims]: https://docs.docker.com/docker-for-mac/#advanced
[snap]: https://snapcraft.io/
[manage docker as a non root user]: https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user
[sudo with kind]: https://github.com/kubernetes-sigs/kind/issues/713#issuecomment-512665315
[docker desktop for windows]: https://hub.docker.com/editions/community/docker-ce-desktop-windows
[switch between windows and linux containers]: https://docs.docker.com/docker-for-windows/#switch-between-windows-and-linux-containers
[version skew]: https://kubernetes.io/docs/setup/release/version-skew-policy/#supported-version-skew
[Quick Start]: /docs/user/quick-start
[AppArmor]: https://en.wikipedia.org/wiki/AppArmor
[firewalld]: https://firewalld.org/
[inotify]: https://en.wikipedia.org/wiki/Inotify

## Docker Desktop for macOS and Windows

Docker containers cannot be executed natively on macOS and Windows, therefore Docker Desktop runs them in a Linux VM. As a consequence, the container networks are not exposed to the host and you cannot reach the kind nodes via IP. You can work around this limitation by configuring [extra port mappings](https://kind.sigs.k8s.io/docs/user/configuration/#extra-port-mappings) though.
