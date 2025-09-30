---
title: "Rootless"
menu:
  main:
    parent: "user"
    identifier: "rootless"
    weight: 3
---
Starting with kind 0.11.0, [Rootless Docker](https://docs.docker.com/go/rootless/), [Rootless Podman](https://github.com/containers/podman/blob/master/docs/tutorials/rootless_tutorial.md) and [Rootless nerdctl](https://github.com/containerd/nerdctl/blob/main/docs/rootless.md) can be used as the node provider of kind.

## Provider requirements

- Docker: 20.10 or later
- Podman: 3.0 or later
- nerdctl: 1.7 or later

## Host requirements

### cgroups v2

The host needs to be running with cgroup v2.
Make sure that the result of the `docker info` command contains `Cgroup Version: 2`.
If it prints `Cgroup Version: 1`, try adding `GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1"` to `/etc/default/grub` and
running `sudo update-grub` to enable cgroup v2.

Your host may also need to enable [cgroup delegation](https://systemd.io/CGROUP_DELEGATION/) for daemon-based controller runtimes.
This is not required for daemonless runtimes, such as podman. Note that this procedure may
[negatively impact performance](https://lists.fedoraproject.org/archives/list/devel@lists.fedoraproject.org/thread/ZMKLS7SHMRJLJ57NZCYPBAQ3UOYULV65/).

To enable cgroup delegation, perform the folowing actions:

1. As root, create the directory `/etc/systemd/system/user@.service.d/` if it does not already exist

   ```sh
   sudo mdkir -p /etc/systemd/system/user@.service.d/
   ```
2. As root, create the file `/etc/systemd/system/user@.service.d/delegate.conf` with the following content:

   ```ini
   [Service]
   Delegate=yes
   ```

3. Reload the systemd daemon:

   ```sh
   sudo systemctl daemon-reload
   ```

4. If using docker, reload the user docker daemon:

   ```sh
   systemctl --user restart docker
   ```

### Networking

Containers running in rootless mode are not typically loaded with host-level iptable modules.
This breaks the behavior of most Ingress and Gateway controllers.

To load the iptable modules into the KinD containers, do the following:

1. As root, create the file `/etc/modules-load.d/iptables.conf` with the following content:

  ```
  ip6_tables
  ip6table_nat
  ip_tables
  iptable_nat
  ```

2. Restart your system to ensure these changes take effect.

### Increase PID Limits

KinD nodes are represented as individual containers on their hosts. Runtimes such as podman set default
[process id limits](https://docs.podman.io/en/v4.3/markdown/options/pids-limit.html#pids-limit-limit)
that may be too low for the node or for a pod running on the node. The NGINX ingress controller is
[particularly susceptible](https://github.com/kubernetes-sigs/kind/issues/3451) to this issue.

To increase the PID limit, do the following:

1. If using podman, edit your `containers.conf` file (generally located in
   `/etc/containers/containers.conf` or `~/.config/containers/containers.conf`) to increase the PIDs
   limit to a desired value (default 4096 on most systems).

    ```ini
    [containers]
    pids_limit = 65536
    ```


### Increase inotify Limits

As documented in [known issues](/docs/user/known-issues/#pod-errors-due-to-too-many-open-files), pods may
fail by reaching inotify watch and instance limits. Ingress controllers such as NGINX and Contour
are particularly susceptible to this issue.

To increase the inotify limits, do the following:

1. As root, create a `.conf` file in `/etc/systctl.d` that increases the `fs.inotify` max user settings:

   ```
   fs.inotify.max_user_watches = 524288
   fs.inotify.max_user_instances = 512
   ```

2. Restart your system for these changes to take effect.


### Allow Unprivileged Binding to HTTP(S) Ports

If you use the `extraPortMappings` method to provide ingress to your KinD cluster, you can allow
the KinD container to bind to ports 80 and 443 on the host. User containers cannot bind to these
ports by default as they are considered privileged.

To allow a KinD node to bind to ports 80 and/or 443 on the host, do the following:

1. As root, create a `.conf` file in `/etc/systctl.d` that lowers the privileged port start number:

   ```
   net.ipv4.ip_unprivileged_port_start=80
   ```

2. Restart your system for these changes to take effect.


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

or

```console
$ systemd-run --scope --user -p "Delegate=yes" kind create cluster
```

If you still get the error `running kind with rootless provider requires setting systemd property "Delegate=yes"` even with [host requirements](#host-requirements) configured.

## Creating a kind cluster with Rootless nerdctl

**Note: containerd v1.7+ is required**

To create a kind cluster with nerdctl, just run:
```console
$ KIND_EXPERIMENTAL_PROVIDER=nerdctl kind create cluster
```

## Tips
- To enable OOM watching, allow `dmesg` by running `sysctl -w kernel.dmesg_restrict=0`.
