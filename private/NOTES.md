


```
git clone https://github.com/SergeAlexandre/kind.git
git tag
git checkout v0.22.0
git switch -c fixeip
```

https://github.com/kubernetes-sigs/kind/issues/2045
https://github.com/kubernetes-sigs/kind/issues/2579



```
make build && ./bin/kind create cluster --config private/sn-config.yaml 

make build && ./bin/kind create cluster --config private/ha-config.yaml

./bin/kind delete cluster
 
docker network rm kind

```


Cleanup:

```
docker stop $(docker ps -a -q)
docker container prune --force
docker volume prune --all --force
docker image prune --all --force
docker network prune --force

```

Single node:
```
docker run --name kind-control-plane --hostname kind-control-plane --label io.x-k8s.kind.role=control-plane --privileged --security-opt seccomp=unconfined --security-opt apparmor=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /lib/modules:/lib/modules:ro -e KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER --detach --tty --label io.x-k8s.kind.cluster=kind --net kind --restart=on-failure:1 --init=false --cgroupns=private --publish=127.0.0.1:52960:6443/TCP -e KUBECONFIG=/etc/kubernetes/admin.conf kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245
```

HA cluster:

```

docker run --name kind-worker2 
--hostname kind-worker2 
--label io.x-k8s.kind.role=worker 
--privileged 
--security-opt seccomp=unconfined 
--security-opt apparmor=unconfined 
--tmpfs /tmp --tmpfs /run 
--volume /var --volume /lib/modules:/lib/modules:ro 
-e KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER 
--detach --tty 
--label io.x-k8s.kind.cluster=kind 
--net kind 
--restart=on-failure:1 
--init=false 
--cgroupns=private 
kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245

docker run --name kind-control-plane2 
--hostname kind-control-plane2 
--label io.x-k8s.kind.role=control-plane 
--privileged 
--security-opt seccomp=unconfined 
--security-opt apparmor=unconfined 
--tmpfs /tmp --tmpfs /run 
--volume /var --volume /lib/modules:/lib/modules:ro 
-e KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER 
--detach --tty 
--label io.x-k8s.kind.cluster=kind 
--net kind 
--restart=on-failure:1 
--init=false 
--cgroupns=private 
--publish=127.0.0.1:53049:6443/TCP 
-e KUBECONFIG=/etc/kubernetes/admin.conf 
kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245

docker run --name kind-external-load-balancer 
--hostname kind-external-load-balancer 
--label io.x-k8s.kind.role=external-load-balancer 
--detach --tty 
--label io.x-k8s.kind.cluster=kind 
--net kind 
--restart=on-failure:1 
--init=false 
--cgroupns=private 
--publish=127.0.0.1:53048:6443/TCP 
docker.io/kindest/haproxy:v20230606-42a2262b

```




# Network segment creation

An ipv6 subnet is created from the network name ('kind'). ipv4 address is deduced

/pkg/cluster/internal/providers/docker/network.go
    ensureNetwork()
        generateULASubnetFromName()

Result: 
172.19.0.0/16

Alternate way: Create the 'kind' network ourself:

docker network create -d=bridge -o com.docker.network.bridge.enable_ip_masquerade=true -o com.docker.network.driver.mtu=65535  --subnet 172.28.0.0/16 --ip-range=172.28.5.0/24 kind

docker network create -d=bridge -o com.docker.network.bridge.enable_ip_masquerade=true -o com.docker.network.driver.mtu=65535  --subnet 172.19.0.0/16 kind
