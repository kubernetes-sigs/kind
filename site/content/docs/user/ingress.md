---
title: "Ingress"
menu:
  main:
    parent: "user"
    identifier: "user-ingress"
    weight: 3
---

# Ingress

This guide covers setting up [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) 
on a kind cluster.

## Setting Up An Ingress Controller

We can leverage KIND's `extraPortMapping` config option when 
creating a cluster to forward ports from the host 
to an ingress controller running on a node. 

We can also setup a custom node label by using `node-labels` 
in the kubeadm `InitConfiguration`, to be used
by the ingress controller `nodeSelector`.


The following ingress controllers are known to work:

 - [Ingress NGINX](#ingress-nginx)

### Ingress NGINX

Create a kind cluster with `extraPortMappings` and `node-labels`.

```shell script
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    apiVersion: kubeadm.k8s.io/v1beta2
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
        authorization-mode: "AlwaysAllow"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
  - containerPort: 443
    hostPort: 443
EOF
```
Apply the [mandatory ingress-nginx components](https://kubernetes.github.io/ingress-nginx/deploy/#prerequisite-generic-deployment-command).

```shell script
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/mandatory.yaml
```
Apply kind specific patches to forward the hostPorts to the 
ingress controller, set taint tolerations and 
schedule it to the custom labelled node.

{{< codefromfile file="examples/ingress/nginx/patch.json" lang="json" >}}

{{% mdwithcopy lang="bash" %}}
kubectl patch deployments -n ingress-nginx nginx-ingress-controller -p '{{< minify file="examples/ingress/nginx/patch.json" >}}' 
{{% /mdwithcopy %}}


Now the Ingress is all setup to be used. 
Refer [Using Ingress](#using-ingress) for a basic example usage.

## Using Ingress

The following example creates simple http-echo services 
and an Ingress object to route to these services.

{{< codefromfile file="examples/ingress/usage.yaml" lang="yaml" >}}

Apply the contents

{{% mdwithcopy lang="bash" %}}
kubectl apply -f {{< absurl "examples/ingress/nginx/example.yaml" >}}
{{% /mdwithcopy %}}

Now verify that the ingress works

{{% mdwithcopy lang="bash" %}}
# should output "foo"
curl localhost/foo
# should output "bar"
curl localhost/bar
{{% /mdwithcopy %}}
