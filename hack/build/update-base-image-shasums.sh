#!/usr/bin/env bash
# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit -o nounset -o pipefail

CONTAINERD_VERSION="1.4.0-106-gce4439a8"

ARCHITECTURES=(
    "amd64"
    "arm64"
    "ppc64le"
)

CONTAINERD_DEFAULT_VERSION="1.4.0-106-gce4439a8"
read -r -p "What version of containerd? (defaults to $CONTAINERD_DEFAULT_VERSION) " CONTAINERD_VERSION
CONTAINERD_VERSION=${CONTAINERD_VERSION:-${CONTAINERD_DEFAULT_VERSION}}

CNI_DEFAULT_VERSION="v0.9.0"
read -r -p "What version of CNI? (defaults to $CNI_DEFAULT_VERSION) " CNI_VERSION
CNI_VERSION=${CNI_VERSION:-${CNI_DEFAULT_VERSION}}

CRICTL_DEFAULT_VERSION="v1.19.0"
read -r -p "What version of crictl: (defaults to $CRICTL_DEFAULT_VERSION) " CRICTL_VERSION
CRICTL_VERSION=${CRICTL_VERSION:-${CRICTL_DEFAULT_VERSION}}

echo
CONTAINERD_BASE_URL="https://github.com/kind-ci/containerd-nightlies/releases/download/containerd-${CONTAINERD_VERSION}"
for ARCH in "${ARCHITECTURES[@]}"; do
    CONTAINERD_URL="${CONTAINERD_BASE_URL}/containerd-${CONTAINERD_VERSION}.linux-${ARCH}.tar.gz.sha256sum"
    SHASUM=$(curl -sSL --retry 5 "${CONTAINERD_URL}" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG CONTAINERD_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
done

echo
# TODO (micahhausler): Once kind-ci/containerd-nightlies adds .sha256sum files, just fetch those
for ARCH in "${ARCHITECTURES[@]}"; do
    RUNC_URL="${CONTAINERD_BASE_URL}/runc.${ARCH}"
    curl -sSL --retry 5 --output "/tmp/runc.${ARCH}" "${RUNC_URL}"
    SHASUM=$(sha256sum "/tmp/runc.${ARCH}" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG RUNC_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
    rm "/tmp/runc.${ARCH}"
done

echo
# TODO (micahhausler): Once https://github.com/kubernetes-sigs/cri-tools/issues/716 is resolved, just fetch the sha256
for ARCH in "${ARCHITECTURES[@]}"; do
    CRICTL_URL="https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-${ARCH}.tar.gz"
    curl -sSL --retry 5 --output "/tmp/crictl.${ARCH}.tgz" "${CRICTL_URL}"
    SHASUM=$(sha256sum "/tmp/crictl.${ARCH}.tgz" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG CRICTL_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
    rm "/tmp/crictl.${ARCH}.tgz"
done

echo
for ARCH in "${ARCHITECTURES[@]}"; do
    CNI_TARBALL="${CNI_VERSION}/cni-plugins-linux-${ARCH}-${CNI_VERSION}.tgz"
    CNI_URL="https://github.com/containernetworking/plugins/releases/download/${CNI_TARBALL}"
    SHASUM=$(curl -sSL --retry 5 "${CNI_URL}.sha256" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG CNI_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
done
