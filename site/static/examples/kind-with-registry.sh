#!/bin/sh
set -o errexit

# new version of kubernetes doesn't allow to use a non tls registry

# 1. Create registry container unless it already exists
reg_name='kind-registry'
reg_port='5001'
cluster_name="${1:-kind}"
if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" != 'true' ]; then
  # create a directory for certificates use to expose the registry
  mkdir -p certs
  
  # create a self-signed certificate
  openssl req \
  -newkey rsa:4096 -nodes -sha256 -keyout certs/domain.key \
  -addext "subjectAltName = DNS:${reg_name}" \
  -subj "/C=EU/ST=State/L=Locality/O=Organization/CN=${reg_name}" \
  -x509 -days 365 -out certs/domain.crt
  
  # allow access to cert from the container
  chmod 755 -R ./certs

  # run the registry
  docker run \
    -d --restart=always -v "$(pwd)"/certs:/certs -p ${reg_port}:443  \
    -e REGISTRY_HTTP_ADDR=0.0.0.0:443 -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt -e REGISTRY_HTTP_TLS_KEY=/certs/domain.key \
    --network bridge --name "${reg_name}" \
    registry:2
fi

# 2. Create kind cluster with containerd registry config for the local registry
# as the certificate is autosigned, we have to disable TLS verification on containerd for the registry
cat <<EOF | kind create cluster --name ${cluster_name} --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."${reg_name}:443"]
          endpoint = ["https://${reg_name}:443"]
      [plugins."io.containerd.grpc.v1.cri".registry.configs]
        [plugins."io.containerd.grpc.v1.cri".registry.configs."${reg_name}:443".tls]
          insecure_skip_verify = true
EOF

# 3. Connect the registry to the cluster network if not already connected
# This allows kind to bootstrap the network but ensures they're on the same network
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' ${reg_name})" = 'null' ]; then
  docker network connect "kind" "${reg_name}"
fi

# 4. Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "${reg_name}:443"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
