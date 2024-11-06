---
title: "Ingress"
menu:
  main:
    parent: "user"
    identifier: "user-ingress"
    weight: 3
description: |-
  This guide covers setting up [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) on a kind cluster.
---
## Setting Up An Ingress Controller

Ingress exposes HTTP and HTTPS routes from outside the cluster to services within the cluster.

1. [Create a cluster](#create-cluster): There are two primary methods to direct external traffic to Services inside the cluster:
    1. using a [LoadBalancer].
    2. leverage KIND's `extraPortMapping` config option when creating a cluster to forward ports from the host.

2. Deploy an Ingress controller, we document [Ingress NGINX](#ingress-nginx) here but other ingresses may work including [Contour](https://projectcontour.io/docs/main/guides/kind/) and Kong, you should follow their docs if you choose to use them.

> **NOTE**: You may also want to consider using [Gateway API](https://gateway-api.sigs.k8s.io/) instead of Ingress.
> Gateway API has an [Ingress migration guide](https://gateway-api.sigs.k8s.io/guides/migrating-from-ingress/).
>
> You can use blixt to test Gateway API with kind https://github.com/kubernetes-sigs/blixt#usage

### Create Cluster

#### Option 1: LoadBalancer

Create a kind cluster and run [Cloud Provider KIND]
to enable the loadbalancer controller which ingress-nginx will use through the loadbalancer API.

{{< codeFromInline lang="bash" >}}
kind create cluster
{{< /codeFromInline >}}

#### Option 2: extraPortMapping

Create a single node kind cluster with `extraPortMappings` to allow the local host to make requests to the Ingress controller over ports 80/443.

{{< codeFromInline lang="bash" >}}
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
{{< /codeFromInline >}}

If you want to run with multiple nodes you must ensure that your ingress-controller is deployed on the same node where you have configured the PortMapping, in this example you can use a [nodeSelector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) to specify the control-plane node name.

{{< codeFromInline lang="yaml" >}}
nodeSelector:
  kubernetes.io/hostname: "kind-control-plane"
{{< /codeFromInline >}}

### Ingress NGINX

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
{{< /codeFromInline >}}

Now the Ingress is all setup. Wait until is ready to process requests running:

{{< codeFromInline lang="bash" >}}
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
{{< /codeFromInline >}}

Refer [Using Ingress](#using-ingress) for a basic example usage.

## Using Ingress

The following example creates simple http-echo services
and an Ingress object to route to these services.

Note, this example uses an nginx-specific Ingress annotation which may not be supported by all Ingress implementations.

```yaml
{{% readFile "static/examples/ingress/usage.yaml" %}}
```

Apply the contents

{{< codeFromInline lang="bash" >}}
kubectl apply -f {{< absURL "examples/ingress/usage.yaml" >}}
{{< /codeFromInline >}}

Now verify that the ingress works

#### Option 1: LoadBalancer

Check the External IP assigned to the Ingress controller by the LoadBalancer

{{< codeFromInline lang="bash" >}}
kubectl -n ingress-nginx get services
NAME                                 TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
ingress-nginx-controller             LoadBalancer   10.96.33.233   192.168.8.5   80:31753/TCP,443:30288/TCP   27d
ingress-nginx-controller-admission   ClusterIP      10.96.80.178   <none>        443/TCP                      27d
{{< /codeFromInline >}}

{{< codeFromInline lang="bash" >}}

# get the loadalancer IP

LOADBALANCER_IP=$(kubectl get services \
   --namespace ingress-nginx \
   ingress-nginx-controller \
   --output jsonpath='{.status.loadBalancer.ingress[0].ip}')

# should output "foo-app"

curl ${LOADBALANCER_IP}/foo

# should output "bar-app"

curl ${LOADBALANCER_IP}/bar
{{< /codeFromInline >}}

#### Option 2: extraPortMapping

The Ingress controller ports will be exposed in your `localhost` address

{{< codeFromInline lang="bash" >}}

# should output "foo-app"

curl localhost/foo

# should output "bar-app"

curl localhost/bar
{{< /codeFromInline >}}

[LoadBalancer]: /docs/user/loadbalancer/
[Cloud Provider KIND]: /docs/user/loadbalancer/
