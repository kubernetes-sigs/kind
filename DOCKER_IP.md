# DockerIP Kind patch

Aim of this patch is to allow building a Kind cluster with a fixed IP address for each container.

The main motivation is to be resilient against Docker restart and reboot of the workstation. 

## What was the problem

The problem occurs only for clusters with several nodes on the control plane. 

Kind rely on Docker to allocate IP addresses when launching a cluster. Such allocation is performed randomly, 
based on the container creation order, witch is also random.

When Docker is stopped (For any reason, including workstation crash) and restarted, the containers can restart without 
any data lost, in most cases. But, IP address are reallocated in containers start ordering, which is different from the 
initial one. So, the kubernetes configuration is invalid.

So this solution to allow container's IP addresses to be explicitly defined.

## Limitation

- Works only with the Docker provider.
- Require manual IP addresses management

## Usage

Of course, first step is to include the patched Kind version in your path. This is transparent, as it behave like 
the standard one by default.

To ensure which one is used, the postfix `.fip` has been added to the base version:

```
kind --version
kind version 0.22.0.fip
```

### Network setting

We will assume here you have a fresh docker installation, without any existing container.

On its first usage, Kind create a docker network named 'kind'. Unfortunately, this default network is created such a 
way (using ipv6) than assigning explicit IP address is refused. 
So, the first step is to recreate it with just an ipv4 subnet. 

```
docker network rm kind # If kind was already used.
docker network create -d=bridge -o com.docker.network.bridge.enable_ip_masquerade=true -o com.docker.network.driver.mtu=65535  --subnet 172.19.0.0/16 kind
```

> The network name `kind` is hard coded and can't be changed.

You can choose whatever subnet you want, provided it is RFC1918 compliant (RFC1918 define addressing for private subnet). 

### Using raw IP

In this patched Kind version, a `dockerIP` property has been added to each node, thus allowing to fix its address. 
(Of course, such address must be included in the subnet defined previously)

Also a `loadBalancer` entry has been added, to allow the same mechanism for this non-kubernetes container.

For example:

```
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ha1
loadBalancer:
  dockerIP: 172.19.101.6
nodes:
  - role: control-plane
    dockerIP: 172.19.101.1
  - role: control-plane
    dockerIP: 172.19.101.2
  - role: control-plane
    dockerIP: 172.19.101.3
  - role: worker
    dockerIP: 172.19.101.4
  - role: worker
    dockerIP: 172.19.101.5
```

If you launch another Kind cluster without `dockerIP` properties, it will be managed the standard way. 
This means IP will be allocated from the beginning of the subnet (First will be 172.19.0.2). Same if you create a non-Kind 
docker container. This is why we began our clusters in the range 172.19.100.0

> Note this is experimental observation and may not be always the case. 

### Using DNS name

Instead of providing raw IP addresses, one can provide DNS name:

```
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ha2
loadBalancer:
  dockerIP: ha2-external-load-balancer
nodes:
  - role: control-plane
    dockerIP: ha2-control-plane
  - role: control-plane
    dockerIP: ha2-control-plane2
  - role: control-plane
    dockerIP: ha2-control-plane3
  - role: worker
    dockerIP: ha2-worker
  - role: worker
    dockerIP: ha2-worker2
```

Such DNS name must resolve to IP addresses in the subnet range. For example, Here is an extract of the corresponding 
`/etc/hosts` file:

```
172.19.102.1 ha2-external-load-balancer
172.19.102.2 ha2-control-plane
172.19.102.3 ha2-control-plane2
172.19.102.4 ha2-control-plane3
172.19.102.5 ha2-worker
172.19.102.6 ha2-worker2
```

We choose to use the container name as the DNS name. But, this is not strictly mandatory in this case. 

### Using `fromDNS` mode

If you look at the previous example, it is obvious there is some redundancy. And you have to manually maintain matching 
between the cluster configuration and the DNS.

To aase management, a new mode has been implemented where each node will find its fixed IP address by resolving its container name.

The mode is activated by adding `dockerIP.fromDNS: true` in the cluster definition. For example:

```
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ha3
dockerIP:
  fromDNS: true
nodes:
  - role: control-plane
  - role: control-plane
  - role: control-plane
  - role: worker
  - role: worker
```

For this to works, the following must be defined in your `/etc/hosts`, or any other DNS subsystem effective on your workstation:

```
172.19.103.1 ha3-external-load-balancer
172.19.103.2 ha3-control-plane
172.19.103.3 ha3-control-plane2
172.19.103.4 ha3-control-plane3
172.19.103.5 ha3-worker
172.19.103.6 ha3-worker2
```

Note than, is such case, you are not free to choose any name. These name are defined by Kind.

Fortunately, they are easily guessable. And error message are explicit for error/retry on cluster creation.

### Node access from the workstation

As the node name are now resolved by our DNS, it can be tempting to connect directly to each node from our workstation. 
But, for Mac users, this is not so simple, as the `kind` network is not exposed on the macOS host.

To overcome this problem, a tool like [docker-mac-net-connect](https://github.com/chipmk/docker-mac-net-connect) can be installed.

