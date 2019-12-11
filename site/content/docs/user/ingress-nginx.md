# Ingress Nginx

Ingress Nginx in kind works by exposing ports `80(http)` and `443(https)`
from the host to the nginx controller using hostPorts.

## Create A Cluster with Ingress Nginx

The following shell script will create a kind cluster deploy 
the standard ingress-nginx components and apply the necessary patches.

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