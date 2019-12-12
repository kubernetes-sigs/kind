#!/bin/sh
set -o errexit

# desired cluster name; default is "kind"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"
REGISTRY_CONTAINER_NAME='kind-registry'
REGISTRY_PORT='5000'

# create registry container unless it already exists
if [ "$(docker inspect -f '{{.State.Running}}' "${REGISTRY_CONTAINER_NAME}")" != 'true' ]; then
  docker run -d -p "${REGISTRY_PORT}:5000" --restart=always --name "${REGISTRY_CONTAINER_NAME}" registry:2
fi

# create a cluster with the local registry enabled in containerd
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches: 
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry:${REGISTRY_PORT}"]
    endpoint = ["http://registry:${REGISTRY_PORT}"]
EOF

# add the registry to /etc/hosts on each node
for node in $(kind get nodes --name "${KIND_CLUSTER_NAME}"); do
  docker exec "${node}" sh -c "echo $(docker inspect --format '{{.NetworkSettings.IPAddress }}' "${REGISTRY_CONTAINER_NAME}") registry >> /etc/hosts"
done