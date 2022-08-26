#!/bin/bash

OCNE_REGISTRY="container-registry.oracle.com/olcne"
OUT_FOLDER=_output

create_tar() {
   local K8S_VERSION=$1
   local PAUSE_VERSION=$2
   local ETCD_VERSION=$3
   local DNS_VERSION=$4
   for IMAGE_NAME in kube-proxy kube-controller-manager kube-scheduler kube-apiserver; do
       IMAGE_VERSION=${K8S_VERSION}
       docker pull ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION}
       docker save ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION} -o ${OUT_FOLDER}/${IMAGE_NAME}.tar
   done

  IMAGE_VERSION=${PAUSE_VERSION}
  IMAGE_NAME="pause"
  docker pull ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION}
  docker save ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION} -o ${OUT_FOLDER}/${IMAGE_NAME}.tar

  IMAGE_VERSION=${ETCD_VERSION}
  IMAGE_NAME="etcd"
  docker pull ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION}
  docker save ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION} -o ${OUT_FOLDER}/${IMAGE_NAME}.tar

  IMAGE_VERSION=${DNS_VERSION}
  IMAGE_NAME="coredns"
  docker pull ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION}
  docker save ${OCNE_REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION} -o ${OUT_FOLDER}/${IMAGE_NAME}.tar
}

# usage create_tar <k8s-ver> <pause-ver> <etcd-ver>
# create_tar v1.23.7 3.6 3.5.1 1.8.6
mkdir -p ${OUT_FOLDER}
create_tar $1 $2 $3 $4