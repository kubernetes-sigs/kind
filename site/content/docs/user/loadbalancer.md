---
title: "LoadBalancer"
menu:
  main:
    parent: "user"
    identifier: "user-loadbalancer"
    weight: 3
description: |-
    This guide covers how to get service of type LoadBalancer working in a kind cluster using [Metallb].

    This guide complements MetalLB [installation docs], and sets up MetalLB using layer2 protocol.  For other protocols check MetalLB [configuration docs].

    With Docker on Linux, you can send traffic directly to the loadbalancer's external IP if the IP space is within the docker IP space.  
    
    On macOS and Windows, docker does not expose the docker network to the host.  Because of this limitation, containers (including kind nodes) are only reachable from the host via port-forwards, however other containers/pods can reach other things running in docker including loadbalancers.  You may want to check out the [Ingress Guide] as a cross-platform workaround.  You can also expose pods and services using extra port mappings as shown in the extra port mappings section of the [Configuration Guide].
    

    [MetalLB]: https://metallb.universe.tf/
    [installation docs]: https://metallb.universe.tf/installation/
    [configuration docs]: https://metallb.universe.tf/configuration/

    [Ingress Guide]: /docs/user/ingress
    [Configuration Guide]: /docs/user/configuration#extra-port-mappings

---

## Installing MetalLB using default manifests

### Apply MetalLB manifest

Since version 0.13.0, MetalLB is configured via CRs and the original way of configuring it via a ConfigMap based configuration
is not working anymore.

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml
{{< /codeFromInline >}}

Wait until the MetalLB pods (controller and speakers) are ready:

{{< codeFromInline lang="bash" >}}
kubectl wait --namespace metallb-system \
                --for=condition=ready pod \
                --selector=app=metallb \
                --timeout=90s
{{< /codeFromInline >}}

### Setup address pool used by loadbalancers

To complete layer2 configuration, we need to provide MetalLB a range of IP addresses it controls.  We want this range to be on the docker kind network.

{{< codeFromInline lang="bash" >}}
docker network inspect -f '{{.IPAM.Config}}' kind
{{< /codeFromInline >}}

If you are using podman 4.0 or higher in rootful mode with the netavark network backend, use the following command instead:
{{< codeFromInline lang="bash" >}}
podman network inspect -f '{{range .Subnets}}{{if eq (len .Subnet.IP) 4}}{{.Subnet}}{{end}}{{end}}' kind
{{< /codeFromInline >}}

The output will contain a cidr such as 172.19.0.0/16.  We want our loadbalancer IP range to come from this subclass.  We can configure MetalLB, for instance, to use 172.19.255.200 to 172.19.255.250 by creating the IPAddressPool and the related L2Advertisement.

```yaml
{{% readFile "static/examples/loadbalancer/metallb-config.yaml" %}}
```

Apply the contents

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://kind.sigs.k8s.io/examples/loadbalancer/metallb-config.yaml
{{< /codeFromInline >}}

## Using LoadBalancer

The following example creates a loadbalancer service that routes to two http-echo pods, one that outputs foo and the other outputs bar.

```yaml
{{% readFile "static/examples/loadbalancer/usage.yaml" %}}
```

Apply the contents

{{< codeFromInline lang="yaml" >}}
kubectl apply -f https://kind.sigs.k8s.io/examples/loadbalancer/usage.yaml
{{< /codeFromInline>}}

Now verify that the loadbalancer works by sending traffic to it's external IP and port.

{{< codeFromInline lang="bash" >}}
LB_IP=$(kubectl get svc/foo-service -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
{{< /codeFromInline >}}

```bash
# should output foo and bar on separate lines 
for _ in {1..10}; do
  curl ${LB_IP}:5678
done
```
