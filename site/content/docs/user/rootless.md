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

### cgroup v2

The host needs to be running with cgroup v2, which is the default for many Linux disributions:

- Ubuntu: 21.10 and later.
- Fedora: 31 and later.
- Arch: April 2021 release and later.

You can verify the cgroup version used by your controller runtime with the following procedure:

- `docker`: Run `docker info` and look for `Cgroup Version: 2` in the output.
- `podman`: Run `podman info` and look for `cgroupVersion: v2` in the output.
- `nerdctl`: Run `nerdctl info` and look for `Cgroup Version: 2` in the output.

If the `info` output prints `Cgroup Version: 1` or equivalent, try the following to enable cgroup v2:

1. In `/etc/default/grub`, add the line `GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1"`
2. Run `sudo update-grub` to enable cgroup v2.

Your host will also need to enable [cgroup delegation](https://systemd.io/CGROUP_DELEGATION/) of the `cpu` controller for
user services. This is enabled by default for distributions running `systemd` version 252 and higher.

To enable cgroup delegation for all the controllers, do the following:

1. Check your version of `systemd` by running `systemctl --version`. If the output prints
   `systemd 252` or higher, no further action is needed. Example output below from a Fedora host:

   ```sh
   $ systemctl --version
   systemd 257 (257.9-2.fc42)
   ```

2. For systems with older versions of `systemd`, first create the directory
   `/etc/systemd/system/user@.service.d/` if it is not present.

   ```sh
   sudo mkdir -p /etc/systemd/system/user@.service.d/
   ```

3. Next, create the file `/etc/systemd/system/user@.service.d/delegate.conf` with the following content:

   ```ini
   [Service]
   Delegate=yes
   ```

4. Reload systemd for these changes to take effect:

   ```sh
   sudo systemctl daemon-reload
   ```

5. If using docker, reload the user docker daemon:

   ```sh
   systemctl --user restart docker
   ```

### Networking

Containers running in rootless mode may not loaded with host-level iptable modules.
This breaks the behavior of most networking components, such as Ingress and Gateway controllers.

To load the iptable modules, do the following:

1. First, use `lsmod` to check which kernel modules are loaded by default for user processes on
   your system. Use `grep` to find which iptable modules are loaded:

   ```sh
   lsmod | grep "ip.*table"
   ```

2. Check the output for the following kernel modules:
   - `ip6_tables`
   - `ip6table_nat`
   - `ip_tables`
   - `iptable_nat`

3. If one or more of the kernel modules above are not present, your system needs to load these at
   startup for each process. First, run the following command to add these missing modules:
   
   ```sh
   sudo tee /etc/modules-load.d/iptables.conf > /dev/null <<'EOF'
   ip6_tables
   ip6table_nat
   ip_tables
   iptable_nat
   EOF
   ```

4. Check that the new module loading configuration is correct. You should see the following output:

   ```sh
   $ cat /etc/modules-load.d/iptables.conf 
   ip6_tables
   ip6table_nat
   ip_tables
   iptable_nat
   ```

5. Next, restart the `systemd-modules-load` service to make these changes effective immediately:

   ```sh
   sudo systemctl restart systemd-modules-load.service
   ```

Alternatively, restart your system to ensure these changes take effect.

### Increase PID Limits

KIND nodes are represented as individual containers on their hosts. Runtimes such as podman set
default [process id limits](https://docs.podman.io/en/v4.3/markdown/options/pids-limit.html#pids-limit-limit)
that may be too low for the node or for a pod running on the node. The Ingress NGINX Controller is
[particularly susceptible](https://github.com/kubernetes-sigs/kind/issues/3451) to this issue.

To increase the PID limit, do the following:

1. If using podman, edit your `containers.conf` file (generally located in
   `/etc/containers/containers.conf` or `~/.config/containers/containers.conf`) to increase the PIDs
   limit to a desired value (default 4096 on most systems):

    ```ini
    [containers]
    pids_limit = 65536
    ```

2. Re-recreate the KIND cluster for these changes to take effect:

   ```sh
   kind delete cluster && kind create cluster
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

2. Reload `sysctl` for these changes to take effect:

   ```sh
   sudo sysctl --system
   ```

Alternatively, restart your system for these changes to take effect.

### Allow Binding to Privileged Ports

If you use the `extraPortMappings` method to provide ingress to your KIND cluster, you can allow
the KIND node container to bind to ports 80 and 443 on the host. User containers cannot bind to
ports below 1024 by default as they are considered privileged.

You can avoid this issue by binding the node to a non-privileged host port, such as 8080 or 8443:

```yaml
# kind config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
```

Note that with this configuration, requests to your cluster ingress will need to add the
appropriate port number. In the example above, HTTP requests must use `localhost:8080` in the URL.

To allow a KIND node to bind to ports 80 and/or 443 on the host, do the following:

1. As root, create a `.conf` file in `/etc/systctl.d` that lowers the privileged port start number:

   ```
   # Allow unprivileged binding to HTTP port 80
   # Use 443 if you only need binding to the default HTTPS port
   net.ipv4.ip_unprivileged_port_start=80
   ```

2. Reload `sysctl` for these changes to take effect:

   ```sh
   sudo sysctl --system
   ```

Alternatively, restart your system for these changes to take effect.

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
