---
title: "Performance"
menu:
  main:
    parent: "user"
    identifier: "performance"
    weight: 2
description: |-
  This guide covers some known performance issues and solutions / workarounds.

  When trying to run KIND you should dimension your host resources accordenly.
  The following resources are specially critical for the cluster performance:

  - disk IO
  - disk space
  - memory
  - CPU
  - inotify limits

  If you notice problems when trying to scale your kind cluster or running
  stressfull workloads, please verify you are not hitting any resource limit.
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

* [Slow Disks](#slow-disks)

* [Pod Errors Due to "too many open files"](#pod-errors-due-to-too-many-open-files) (likely [inotify] limits which are not namespaced)


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

