---
title: "Rootless"
menu:
  main:
    parent: "user"
    identifier: "rootless"
    weight: 3
---
Starting with kind 0.11.0, [Rootless Docker](https://docs.docker.com/go/rootless/) and [Rootless Podman](https://github.com/containers/podman/blob/master/docs/tutorials/rootless_tutorial.md) can be used as the node provider of kind.

## Provider requirements
- Docker: 20.10 or later
- Podman: 3.0 or later

## Host requirements
The host needs to be running with cgroup v2.
Make sure that the result of the `docker info` command contains `Cgroup Version: 2`.
If it prints `Cgroup Version: 1`, try adding `GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1"` to `/etc/default/grub` and
running `sudo update-grub` to enable cgroup v2.

Also, depending on the host configuration, the following steps might be needed:

- Create `/etc/systemd/system/user@.service.d/delegate.conf` with the following content, and then run `sudo systemctl daemon-reload`:

  ```ini
  [Service]
  Delegate=yes
  ```

  (This is not enabled by default because ["the runtime impact of
  [delegating the "cpu" controller] is still too
  high"](https://lists.fedoraproject.org/archives/list/devel@lists.fedoraproject.org/thread/ZMKLS7SHMRJLJ57NZCYPBAQ3UOYULV65/).
  Beware that changing this configuration may affect system
  performance.)

  Please note that:

  - `/etc/systemd/system/user@.service.d/` directory needs to be created if not already present on your host
  - If using Docker and it was already running when this step was done, a restart is needed for the changes to take
    effect
      {{< codeFromInline lang="bash" >}}
      systemctl --user restart docker
      {{< /codeFromInline >}}

- Create `/etc/modules-load.d/iptables.conf` with the following content:

  ```
  ip6_tables
  ip6table_nat
  ip_tables
  iptable_nat
  ```

## Restrictions

The restrictions of Rootless Docker apply to kind clusters as well.

e.g.
- OverlayFS cannot be used unless the host is using kernel >= 5.11, or Ubuntu/Debian kernel
- Cannot mount block storage
- Cannot mount NFS

## Creating a kind cluster with Rootless Docker

To create a kind cluster with Rootless Docker, just run:
```console
$ export DOCKER_HOST=unix://${XDG_RUNTIME_DIR}/docker.sock
$ kind create cluster
```

## Creating a kind cluster with Rootless Podman

To create a kind cluster with Rootless Podman, just run:
```console
$ KIND_EXPERIMENTAL_PROVIDER=podman kind create cluster
```

On some distributions, you might need to use systemd-run to start kind into its own cgroup scope:
```console
$ systemd-run --scope --user kind create cluster
```

## Tips
- To enable OOM watching, allow `dmesg` by running `sysctl -w kernel.dmesg_restrict=0`.
