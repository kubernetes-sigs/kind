


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


# Network segment creation

An ipv6 subnet is created from the network name ('kind'). ipv4 address is deduced

/pkg/cluster/internal/providers/docker/network.go
    ensureNetwork()
        generateULASubnetFromName()


Alternate way: Create the 'kind' network ourself:

docker network create -d=bridge -o com.docker.network.bridge.enable_ip_masquerade=true -o com.docker.network.driver.mtu=65535  --subnet 172.28.0.0/16 --ip-range=172.28.5.0/24 kind
