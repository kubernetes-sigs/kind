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

We can leverage KIND's `extraPortMapping` config option when
creating a cluster to forward ports from the host
to an ingress controller running on a node.

We can also setup a custom node label by using `node-labels`
in the kubeadm `InitConfiguration`, to be used
by the ingress controller `nodeSelector`.


1. [Create a cluster](#create-cluster)
2. Deploy an Ingress controller, the following ingress controllers are known to work:
    - [Contour](#contour)
    - [Ingress Kong](#ingress-kong)
    - [Ingress NGINX](#ingress-nginx)

### Create Cluster

Create a kind cluster with `extraPortMappings` and `node-labels`.

- **extraPortMappings** allow the local host to make requests to the Ingress controller over ports 80/443
- **node-labels** only allow the ingress controller to run on a specific node(s) matching the label selector

{{< codeFromInline lang="bash" >}}
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
{{< /codeFromInline >}}

### Contour

Deploy [Contour components](https://projectcontour.io/quickstart/contour.yaml).

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
{{< /codeFromInline >}}

Apply kind specific patches to forward the hostPorts to the
ingress controller, set taint tolerations and
schedule it to the custom labelled node.

```json
{{% readFile "static/examples/ingress/contour/patch.json" %}}
```

Apply it by running:

{{< codeFromInline lang="bash" >}}
kubectl patch daemonsets -n projectcontour envoy -p '{{< minify file="static/examples/ingress/contour/patch.json" >}}'
{{< /codeFromInline >}}

Now the Contour is all setup to be used.
Refer to [Using Ingress](#using-ingress) for a basic example usage.

Additional information about Contour can be found at: [projectcontour.io](https://projectcontour.io)

### Ingress Kong

Deploy [Kong Ingress Controller (KIC)](https://docs.konghq.com/kubernetes-ingress-controller/2.1.x/concepts/design/).

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://raw.githubusercontent.com/Kong/kubernetes-ingress-controller/master/deploy/single/all-in-one-dbless.yaml
{{< /codeFromInline >}}

Apply kind specific patches to forward the `hostPorts` to the ingress controller, set taint tolerations, and schedule it to the custom labeled node.

```json
{{% readFile "static/examples/ingress/kong/deployment.patch.json" %}}
```

Apply it by running:

{{< codeFromInline lang="bash" >}}
kubectl patch deployment -n kong proxy-kong -p '{{< minify file="static/examples/ingress/kong/deployment.patch.json" >}}'
{{< /codeFromInline >}}

Apply kind specific patch to change service type to `NodePort`:

```json
{{% readFile "static/examples/ingress/kong/service.patch.json" %}}
```

Apply it by running:

{{< codeFromInline lang="bash" >}}
kubectl patch service -n kong kong-proxy -p '{{< minify file="static/examples/ingress/kong/service.patch.json" >}}'
{{< /codeFromInline >}}

KIC can be used to configure ingress now.

You can try the example in [Using Ingress](#using-ingress) at this moment,
but KIC will not automatically handle `Ingress` object defined there.
`Ingress` resources must include `ingressClassName: kong` under `spec` of `Ingress`  for being controlled by Kong Ingress Controller (it will be ignored otherwise).
So once the example has been loaded, you can add this annotation with:

{{< codeFromInline lang="bash" >}}
kubectl patch ingress example-ingress -p '{"spec":{"ingressClassName":"kong"}}'
{{< /codeFromInline >}}

Refer [Using Ingress](#using-ingress) for primary example usage.


### Ingress NGINX

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
{{< /codeFromInline >}}

The manifests contains kind specific patches to forward the hostPorts to the
ingress controller, set taint tolerations and schedule it to the custom labelled node.

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

```yaml
{{% readFile "static/examples/ingress/usage.yaml" %}}
```

Apply the contents

{{< codeFromInline lang="bash" >}}
kubectl apply -f {{< absURL "examples/ingress/usage.yaml" >}}
{{< /codeFromInline >}}

Now verify that the ingress works

{{< codeFromInline lang="bash" >}}
# should output "foo-app"
curl localhost/foo/hostname
# should output "bar-app"
curl localhost/bar/hostname
{{< /codeFromInline >}}
