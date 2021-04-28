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
    - [Ambassador](#ambassador)
    - [Contour](#contour)
    - [Ingress NGINX](#ingress-nginx)
    - [Kong](#kong)

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

### Ambassador

[Ambassador](https://www.getambassador.io/) will be installed with the help of
the [Ambassador operator](https://www.getambassador.io/docs/latest/topics/install/aes-operator/).

First install the CRDs with

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://github.com/datawire/ambassador-operator/releases/latest/download/ambassador-operator-crds.yaml
{{< /codeFromInline >}}

Now install the kind-specific manifest for installing Ambassador with the operator
in the `ambassador` namespace:

{{< codeFromInline lang="bash" >}}
kubectl apply -n ambassador -f https://github.com/datawire/ambassador-operator/releases/latest/download/ambassador-operator-kind.yaml
kubectl wait --timeout=180s -n ambassador --for=condition=deployed ambassadorinstallations/ambassador
{{< /codeFromInline >}}

Ambassador is now ready for use. You can try the example in [Using Ingress](#using-ingress) at this moment,
but Ambassador will not automatically load the `Ingress` defined there. `Ingress` resources must include
the annotation `kubernetes.io/ingress.class: ambassador` for being recognized by Ambassador (otherwise they are just ignored).
So once the example has been loaded you can add this annotation with:

{{< codeFromInline lang="bash" >}}
kubectl annotate ingress example-ingress kubernetes.io/ingress.class=ambassador
{{< /codeFromInline >}}

Ambassador should be exposing your Ingress now. Please find additional documentation on
Ambassador [here](https://www.getambassador.io/docs/latest/).

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


### Ingress NGINX

{{< codeFromInline lang="bash" >}}
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/kind/deploy.yaml
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

### Kong

To run Kong ingress, create your kind cluster with port mappings to
80/443:

```yaml
cat <<EOF | kind create cluster --name my-kind-cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 80
    protocol: TCP
  - containerPort: 30443
    hostPort: 443
    protocol: TCP
EOF
```

Next, create a ClusterRoleBinding granting full cluster admin
privileges to the `default` ServiceAccount in `kube-system`:

```bash
kubectl create clusterrolebinding add-on-cluster-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=kube-system:default
```

Next, install the [Kong helm chart](https://github.com/Kong/charts),
setting the following values:

```yaml
proxy:
  type: NodePort
  http:
    nodePort: 30080
  tls:
    nodePort: 30443
```

Or, if you prefer, you can give these options on the command line:

```bash
helm install kong kong/kong \
  --set proxy.type=NodePort \
  --set proxy.http.nodePort=30080 \
  --set proxy.tls.nodePort=30443
```

Kong should be functioning as your Ingress at this point.

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
# should output "foo"
curl localhost/foo
# should output "bar"
curl localhost/bar
{{< /codeFromInline >}}
