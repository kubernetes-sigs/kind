---
title: "Accesing applications inside the cluster"
menu:
  main:
    parent: "user"
    identifier: "exposing-applications"
    weight: 1
---
# Accesing services and applications inside a kubernetes cluster

If you need a kubernetes cluster to develop or to test your applications, services, charts, ... Minikube is a great tool but lacks the capability to create multi-node clusters.

`kind` use docker to create to create multi-nodes cluster, despite the great benefits of docker it has the problem that the implementation differ for each OS, and [Windows][[Docker for Windows networking]] and [Macs][Docker for Mac networking] have important networking limitations. Main issue is that you can't reach the cluster node internal ip addresses.

This guide covers how to access your pods and services inside your kubernetes cluster once you have deployed them with `kind`. We really suggest you to get familiar with the [kubernetes concepts for exposing applications][kubernetes exposing services] and [when to use what][kubernetes external access].

For Linux users you can follow this nice write up done by @mauilion using [MetalLB Load Balancer][kind metallb].

The rest of the document will be based on Mac OSX and Windows with WSL (Windows Subsystem for Linux)

## Create the Kubernetes cluster

Use a configuration file to create the kuberntes cluster 
`config-ipv4-ci.yaml`

```yaml
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
# the control plane node
- role: control-plane
- role: worker
- role: worker
````

And create the cluster

```sh 
$ kind create cluster --config config-ipv4-ci.yaml
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.14.1) 🖼
 ✓ Preparing nodes 📦📦📦
 ✓ Creating kubeadm config 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
 ✓ Joining worker nodes 🚜
Cluster creation complete. You can now use the cluster with:

export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
kubectl cluster-info
```

## Expose an application

Let's take a widely used [example][kubernetes access application] to access applications and apply it to `kind`.

Run a hello-world application

```sh
kubectl run hello-world --replicas=2 --labels="run=load-balancer-example" --image=gcr.io/google-samples/node-hello:1.0  --port=8080
```

and wait until it's ready:

```sh
kubectl wait --for=condition=available --timeout=600s  deployment/hello-world

deployment.extensions/hello-world condition met
```

```sh
kubectl get deployments hello-world

NAME          READY   UP-TO-DATE   AVAILABLE   AGE
hello-world   2/2     2            2           94s
```

Once we have our application deployed we can expose it using a Service object

### ClusterIP

> Exposes the service on a cluster-internal IP. Choosing this value makes the service only reachable from within the cluster. This is the default ServiceType

```sh
kubectl expose deployment hello-world --port=80 --target-port=8080 --name=hello-world

service/hello-world exposed
```

```
kubectl describe services hello-world

Name:              hello-world
Namespace:         default
Labels:            run=load-balancer-example
Annotations:       <none>
Selector:          run=load-balancer-example
Type:              ClusterIP
IP:                10.106.26.187
Port:              <unset>  80/TCP
TargetPort:        8080/TCP
Endpoints:         10.36.0.1:8080,10.44.0.1:8080
Session Affinity:  None
Events:            <none>
```

### NodePort

> NodePort, as the name implies, opens a specific port on all the Nodes (the VMs), and any traffic that is sent to this port is forwarded to the service.


```
kubectl expose deployment hello-world --type=NodePort --name=hello-world-nodeport

service/hello-world-nodeport exposed
```

```
kubectl describe services hello-world-nodeport
Name:                     hello-world-nodeport
Namespace:                default
Labels:                   run=load-balancer-example
Annotations:              <none>
Selector:                 run=load-balancer-example
Type:                     NodePort
IP:                       10.96.59.230
Port:                     <unset>  8080/TCP
TargetPort:               8080/TCP
NodePort:                 <unset>  30039/TCP
Endpoints:                10.36.0.1:8080,10.44.0.1:8080
Session Affinity:         None
External Traffic Policy:  Cluster
Events:                   <none>
```


### Nginx ingress controller

> Ingress is actually NOT a type of service. Instead, it sits in front of multiple services and act as a “smart router” or entrypoint into your cluster.

TODO

## Accesing the application

For some of this methods we'll need the cluster IP addresses

```
kubectl get nodes -o wide

NAME                 STATUS   ROLES    AGE     VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE                                  KERNEL-VERSION     CONTAINER-RUNTIME
kind-control-plane   Ready    master   7m1s    v1.14.1   172.17.0.4    <none>        Ubuntu Disco Dingo (development branch)   4.9.125-linuxkit   containerd://1.2.6-0ubuntu1
kind-worker          Ready    <none>   5m58s   v1.14.1   172.17.0.2    <none>        Ubuntu Disco Dingo (development branch)   4.9.125-linuxkit   containerd://1.2.6-0ubuntu1
kind-worker2         Ready    <none>   5m59s   v1.14.1   172.17.0.3    <none>        Ubuntu Disco Dingo (development branch)   4.9.125-linuxkit   containerd://1.2.6-0ubuntu1
```

### Kubectl port-forward

This is the most simple one and well documented

https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/

### Port forwarding with socat

Docker allows to port map containers ports to our localhost, the problem is that this port mapping can be configured only when we create the container. If we want to map new port we have to recreate the container.

We can leverage this functionality to create one container that allows to forward ports, this container will forward the new ports mapped to the corresponding ports on the cluster.

This process is well documented on different places, ie.

https://sosedoff.com/2018/04/25/expose-docker-ports.html

https://andrewwilkinson.wordpress.com/2017/11/22/exposing-docker-ports-after-the-fact/




### Socks Proxy

> A SOCKS server is a general purpose proxy server that establishes a TCP connection to another server on behalf of a client, then routes all the traffic back and forth between the client and the server. It works for any kind of network protocol on any port

Instead of having to recreate the port mapping constantly we can create a socks proxy inside the docker network and use it to connect to the exposed applications.

SOCKS proxy can be used with any web browser, just have to configure your application to use it and use your service IP and Port target directly.

Create a container with a SOCKS proxy and forward the port

```sh
docker run --rm -d --privileged --name socks5 -p 1080:1080 aojea/socks5
```

Accessing the NodePort service, for that we need only the IP of one of the cluster nodes:

```sh
curl --socks5 localhost:1080 172.17.0.4:30039
Hello Kubernetes!
```

We can reach the service IP too, for that we have to add the routes to the service Subnet through one of the cluster nodes. By default `kubeadm` uses "10.96.0.0/12" but it can changed.

```sh
docker exec -it socks5 sh
# add the ip route through the kind-control-plane node
ip route add 10.96.0.0/12 via 172.17.0.4
```  

And now we can access the application using the the Service IP and Service port:

```sh
curl --socks5 localhost:1080  10.96.59.230:8080
Hello Kubernetes!
```


### tun2socks

This is the most complex option but with a better UX, it consists in creating a tun interface in the host and route all traffic that belongs to the k8s cluster through it.

Internally it uses the SOCKS proxy server that we previously deployed in the docker network and forwarded to our host.

Some example implementations:

https://github.com/yinghuocho/gotun2socks

https://github.com/FlowerWrong/tun2socks












[kind metallb]: https://mauilion.dev/posts/kind-metallb/
[kubernetes external access]: https://medium.com/google-cloud/kubernetes-nodeport-vs-loadbalancer-vs-ingress-when-should-i-use-what-922f010849e0
[kubernetes access application]: https://kubernetes.io/docs/tutorials/stateless-application/expose-external-ip-address/
[kubernetes exposing services]: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
[Docker for Mac networking]: https://docs.docker.com/docker-for-mac/networking/
[Docker for Windows networking]: https://docs.docker.com/docker-for-windows/networking/