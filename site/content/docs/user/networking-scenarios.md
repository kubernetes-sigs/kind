---
title: "Using KIND to emulate complex network scenarios"
menu:
  main:
    parent: "user"
    identifier: "networking-scenarios"
    weight: 3
---
# Using KIND to emulate complex network scenarios [Linux Only]

KIND runs Kubernetes cluster in Docker, and leverages Docker networking for all the network features: portmapping, IPv6, containers connectivity, ...

## Docker Networking

<img src="/docs/user/images/kind-docker-network.png"/>

KIND uses [the default docker bridge network](https://docs.docker.com/network/bridge/#use-the-default-bridge-network).

It creates a bridge named **docker0** 

{{< codeFromInline lang="bash" >}}
$ docker network ls
NETWORK ID          NAME                DRIVER              SCOPE
8fb3fa672192        bridge              bridge              local
0c8d84f52592        host                host                local
558684a8afb8        none                null                local
{{< /codeFromInline >}}

with IP address 172.17.0.1/16.

{{< codeFromInline lang="bash" >}}
$ ip addr show docker0
4: docker0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default 
    link/ether 02:42:83:eb:5e:67 brd ff:ff:ff:ff:ff:ff
    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0
       valid_lft forever preferred_lft forever
{{< /codeFromInline >}}

Docker also creates iptables NAT rules on the docker host that masquerade the traffic from the containers connected to docker0 bridge to connect to the outside world.

## Kubernetes Networking

<img src="/docs/user/images/kind-kubernetes-network-kindnet.png"/>

[The Kubernetes network model](https://kubernetes.io/docs/concepts/cluster-administration/networking/#the-kubernetes-network-model) implies end to end connectivity without NAT between Pods.

By default, KIND uses its own CNI plugin, **Kindnet**, that install the corresponding routes and iptables rules on the cluster nodes.

## Multiple clusters

As we explained before, all KIND clusters are sahring the same docker network, that means that all the cluster nodes have direct connectivity.

If we want to spawn multiple cluster and provide Pod to Pod connectivity between different clusters, first we have to configure the cluster networking parameters to avoid address overlapping.

### Example: Kubernetes multi-region

Let's take an example emulating 2 clusters: A and B.

For cluster A we are going to use the following network parameters:

{{< codeFromInline lang="bash" >}}
cat <<EOF | kind create cluster --name clusterA --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  podSubnet: "10.110.0.0/16"
  serviceSubnet: "10.115.0.0/16"
nodes:
- role: control-plane
- role: worker
EOF
{{< /codeFromInline >}}

And Cluster B:

{{< codeFromInline lang="bash" >}}
cat <<EOF | kind create cluster --name clusterB --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  podSubnet: "10.220.0.0/16"
  serviceSubnet: "10.225.0.0/16"
nodes:
- role: control-plane
- role: worker
EOF
{{< /codeFromInline >}}

All the nodes in each cluster will have routes to the podsSubnets assigned to the nodes of the same cluster.
If we want to provide Pod to Pod connectivity between different clusters we just have to do the same in each node:

{{< codeFromInline lang="bash" >}}
for each node in all clusters
  for each different node in all clusters
    obtain pod subnet assigned to the node
    obtain ip address in the docker network
    install route to the pod subnet via the node IP
{{< /codeFromInline >}}

### Example: Emulate external VMs

By default Docker will attach all containers to the **docker0** bridge:

{{< codeFromInline lang="bash" >}}
$ docker run -d --name alpine alpine tail -f /dev/null
8b94e9dabea847c004ce9fd7a69cdbc82eb93e31857c25c0a8872706efb08a4d
$ docker exec -it alpine ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
10: eth0@if11: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP 
    link/ether 02:42:ac:11:00:03 brd ff:ff:ff:ff:ff:ff
    inet 172.17.0.3/16 brd 172.17.255.255 scope global eth0
{{< /codeFromInline >}}

That means that Pods will be able to reach other dockers containers that does not belong to any KIND cluster, however, the docker container will not be able to answer to the Pod IP address until we intall the correspoding routes.

We can solve it installing routes in the new containers to the Pod Subnets in each Node.

### Example: Multiple network interfaces

There are scenarios that will require to create multiple interfaces in the KIND nodes to test multi-homing, VLANS, CNI plugins, ... 

In order to do that, you can use tools like [koko](https://github.com/redhat-nfvpe/koko) to create new networking interfaces on the KIND nodes, you can check several examples of creating complex topologies with containers in this repo https://github.com/aojea/frr-lab.