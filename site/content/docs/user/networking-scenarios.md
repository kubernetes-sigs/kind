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

## Services

[Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/) are an abstract way to expose an application running on a set of Pods as a network service.

The are different types of Services:

* Cluster IP
* NodePort
* LoadBalancer
* Headless
* ExternalName

In Linux hosts, you can access directly the Cluster IP address of the services just adding one route to the configured **serviceSubnet** parameters via any of the nodes that belong to the cluster, so there is no need to use NodePort or LoadBalancer services.

## Multiple clusters

As we explained before, all KIND clusters are sharing the same docker network, that means that all the cluster nodes have direct connectivity.

If we want to spawn multiple clusters and provide Pod to Pod connectivity between different clusters, first we have to configure the cluster networking parameters to avoid address overlapping.

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

That means that Pods will be able to reach other Docker containers that does not belong to any KIND cluster, however, the Docker container will not be able to answer to the Pod IP address until we install the correspoding routes.

We can solve it installing routes in the new containers to the Pod Subnets in each Node.

### Example: Multiple network interfaces and Multi-Home Nodes

There can be scenarios that requite multiple interfaces in the KIND nodes to test multi-homing, VLANS, CNI plugins, ... 

Typically, you will want to use loopback addresses for communication. We can configure those loopback addresses after the cluster has been created, and then modify the Kubernetes components to use them.

When creating the cluster we must add the loopback IP address of the control plane to the certificate SAN (the apiserver binds to "all-interfaces" by default):

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
# add the loopback to apiServer cert SANS
kubeadmConfigPatchesJSON6902:
- group: kubeadm.k8s.io
  kind: ClusterConfiguration
  patch: |
    - op: add
      path: /apiServer/certSANs/-
      value: my-loopback
```

In order to create the network interfaces, you can use tools like [koko](https://github.com/redhat-nfvpe/koko) to create new networking interfaces on the KIND nodes, you can check several examples of creating complex topologies with containers in this repo https://github.com/aojea/frr-lab.

Other alternative is [using Docker user defined bridges](https://docs.docker.com/network/bridge/#connect-a-container-to-a-user-defined-bridge):

```sh
LOOPBACK_PREFIX="1.1.1."
MY_BRIDGE="my_net2"
MY_ROUTE=10.0.0.0/24
MY_GW=172.16.17.1
# Create 2nd network
docker network create ${MY_BRIDGE}
# Configure nodes to use the second network
for n in $(kind get nodes); do
  # Connect the node to the second network
  docker network connect ${MY_BRIDGE} ${n}
  # Configure a loopback address
  docker exec ${n} ip addr add ${LOOPBACK_PREFIX}${i}/32 dev lo
  # Add static routes
  docker exec ${n} ip route add ${MY_ROUTE} via {$MY_GW}
done
```

After the cluster has been created, we have to modify, in the control-plane node,  the kube-apiserver `--advertise-address` flag in the static pod manifest in `/etc/kubernetes/manifests/kube-apiserver.yaml` (once you write the file it restarts the pod with the new config):

```yaml
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  labels:
    component: kube-apiserver
    tier: control-plane
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-apiserver
    - --advertise-address=172.17.0.4
```

and then change in all the nodes the kubelet `node-ip` flag:

```
root@kind-worker:/# more /var/lib/kubelet/kubeadm-flags.env 
KUBELET_KUBEADM_ARGS="--container-runtime=remote --container-runtime-endpoint=/run/containerd/containerd.sock --fail-swap-on=false --node-ip=172.17.0.4"
 ```

and restart them `systemctl restart kubelet` to use the new config

