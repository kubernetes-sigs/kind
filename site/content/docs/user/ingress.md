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
forward ports from the host to an ingress controller running on a node.
The following ingress controllers are known to work:

 - [Ingress NGINX](#ingress-nginx)

### Ingress NGINX

The following shell script will create a kind cluster, deploy 
the standard ingress-nginx components, and apply the necessary patches.

**Note:** This setup is independent of the number of worker nodes 
and only works on a single control-plane cluster.

```bash 
#!/bin/sh
set -o errexit

# Create cluster with hostPorts opened for http and https
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
  - containerPort: 443
    hostPort: 443
EOF

# Apply the mandatory ingress-nginx components 
# https://kubernetes.github.io/ingress-nginx/deploy/#prerequisite-generic-deployment-command
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/mandatory.yaml

# Apply kind specific patches
kubectl patch deployments -n ingress-nginx nginx-ingress-controller -p "$(cat<<EOF
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
        node-role.kubernetes.io/master: ''
      tolerations:
      # tolerate the the control-plane taints
      - key: node-role.kubernetes.io/master
        operator: Equal
        effect: NoSchedule
EOF
)"
```


Now you will want to checkout [Using Ingress](#using-ingress)


## Using Ingress

The following example creates simple http-echo services 
and an Ingress object to route to these services.

```bash
cat <<EOF | kubectl apply -f -
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
EOF
```

Now verify that the ingress works

```bash
curl localhost/foo # should output "foo"
curl localhost/bar # should output "bar"
```
