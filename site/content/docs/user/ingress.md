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
## Compatibility：
This guide applies to [cloud-provider-kind](https://github.com/kubernetes-sigs/cloud-provider-kind) v0.9.0+. For older versions, refer to historical docs.

## Setting Up Ingress

Ingress exposes HTTP and HTTPS routes from outside the cluster to services within the cluster.

Since cloud-provider-kind v0.9.0, it natively supports Ingress. No third-party ingress controllers are required by default.

For third-party ingress solutions (e.g., Ingress NGINX, Contour), please follow their official documentation.

> **NOTE**: Gateway API is also natively supported (along with Ingress). See the official [Ingress migration guide](https://gateway-api.sigs.k8s.io/guides/migrating-from-ingress/) for details.

## Create Cluster

The built-in cloud provider automatically enables LoadBalancer support for Ingress. Create a cluster as follows.

{{< codeFromInline lang="bash" >}}
kind create cluster
{{< /codeFromInline >}}

## Using Ingress

The following example creates simple http-echo services and an Ingress object to route to these services.

```yaml
{{% readFile "static/examples/ingress/usage.yaml" %}}
```

Apply the configuration:

{{< codeFromInline lang="bash" >}}
kubectl apply -f {{< absURL "examples/ingress/usage.yaml" >}}
{{< /codeFromInline >}}

### Verify Ingress Works

Check the External IP assigned to the Ingress by the built-in LoadBalancer.

{{< codeFromInline lang="bash" >}}
kubectl get ingress
NAME              CLASS     HOSTS         ADDRESS        PORTS   AGE
example-ingress   <none>    example.com   172.18.0.5     80      10m
{{< /codeFromInline >}}

{{< codeFromInline lang="bash" >}}
# get the Ingress IP

INGRESS_IP=$(kubectl get ingress example-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# should output "foo-app"

curl ${INGRESS_IP}/foo

# should output "bar-app"
curl ${INGRESS_IP}/bar
{{< /codeFromInline >}}

[LoadBalancer]: /docs/user/loadbalancer/
[Cloud Provider KIND]: /docs/user/loadbalancer/
