---
title: "Running kind with Rootless Docker"
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

cgroup v2 is enabled by default on Fedora.
On other distros, cgroup v2 can be typically enabled by adding `GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1"` to `/etc/default/grub` and
running `sudo update-grub`.

Also, depending on the host configuration, the following steps might be needed:

- Create `/etc/systemd/system/user@.service.d/delegate.conf` with the following content, and then run `sudo systemctl daemon-reload`:
```ini
[Service]
Delegate=yes
```

- Create `/etc/modules-load.d/iptables.conf` with the following content:
```
iptables_nat
ip6tables_nat
```

## Restrictions

The restrictions of Rootless Docker apply to kind clusters as well.

e.g.
- OverlayFS cannot be used unless the host is using kernel >= 5.11, or Ubuntu/Debian kernel
- Cannot mount block storages
- Cannot mount NFS

## Creating a kind cluster with Rootless Docker

To create a kind cluster with Rootless Docker, just run:
```console
$ export DOCKER_HOST=unix://${XDG_RUNTIME_DIR}/docker.sock
$ kind create cluster
```

To create a kind cluster with Rootless Podman, just run:
```console
$ KIND_EXPERIMENTAL_PROVIDER=podman kind create cluster
```

## Tips
- To enable OOM watching, allow `dmesg` by running `sysctl -w kernel.dmesg_restrict=0`.
