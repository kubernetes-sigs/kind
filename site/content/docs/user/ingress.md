---
title: "Ingress"
menu:
  main:
    parent: "user"
    identifier: "user-ingress"
    weight: 3
---

## Setting Up An Ingress Controller

We can leverage KIND's `extraPortMapping` config option when creating a cluster to
forward ports from the host to an ingress controller running on a node. We can also specify 
custom node label by using `node-labels` in the kubeadm `InitConfiguration`, to be used
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
Apply kind specific patches

```yaml
spec:
  template:
    spec:
      containers:
      - name: nginx-ingress-controller
        ports:
        - containerPort: 80
        # Proxy the host port 80 for http
          hostPort: 80
        - containerPort: 443
        # Proxy the host port 443 for https
          hostPort: 443
      nodeSelector:
        # schedule it on the control-plane node
        ingress-ready: 'true'
      tolerations:
      # tolerate the the control-plane taints
      - key: node-role.kubernetes.io/master
        operator: Equal
        effect: NoSchedule
```

```shell script
kubectl patch deployments -n ingress-nginx nginx-ingress-controller -p "$(curl https://kind.sigs.k8s.io/manifests/ingress/nginx/patch.yaml)"
```


Now you will want to checkout [Using Ingress](#using-ingress)


## Using Ingress

The following example creates simple http-echo services 
and an Ingress object to route to these services.

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: foo-app
  labels:
    app: foo
spec:
  containers:
    - name: foo-app
      image: hashicorp/http-echo
      args:
        - "-text=foo"
---
kind: Service
apiVersion: v1
metadata:
  name: foo-service
spec:
  selector:
    app: foo
  ports:
    - port: 5678 # Default port used by the image
---
kind: Pod
apiVersion: v1
metadata:
  name: bar-app
  labels:
    app: bar
spec:
  containers:
    - name: bar-app
      image: hashicorp/http-echo
      args:
        - "-text=bar"
---
kind: Service
apiVersion: v1
metadata:
  name: bar-service
spec:
  selector:
    app: bar
  ports:
    - port: 5678 # Default port used by the image
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
    ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - http:
      paths:
        - path: /foo
          backend:
            serviceName: foo-service
            servicePort: 5678
        - path: /bar
          backend:
            serviceName: bar-service
            servicePort: 5678
---
```

Apply the contents

```shell script
kubectl apply -f https://kind.sigs.k8s.io/manifests/ingress/nginx/example.yaml
```

Now verify that the ingress works

```shell script
curl localhost/foo # should output "foo"
curl localhost/bar # should output "bar"
```
