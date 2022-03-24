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

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"

# get the versions from the dockerfile
CONTAINERD_VERSION="$(sed -n 's/ARG CONTAINERD_VERSION="\(.*\)"/\1/p' ./images/base/Dockerfile)"
CNI_PLUGINS_VERSION="$(sed -n 's/ARG CNI_PLUGINS_VERSION="\(.*\)"/\1/p' ./images/base/Dockerfile)"
CRICTL_VERSION="$(sed -n 's/ARG CRICTL_VERSION="\(.*\)"/\1/p' ./images/base/Dockerfile)"
CONTAINERD_FUSE_OVERLAYFS_VERSION="$(sed -n 's/ARG CONTAINERD_FUSE_OVERLAYFS_VERSION="\(.*\)"/\1/p' ./images/base/Dockerfile)"

# darwin is great
SED="sed"
if which gsed &>/dev/null; then
  SED="gsed"
fi
if ! (${SED} --version 2>&1 | grep -q GNU); then
  echo "!!! GNU sed is required.  If on OS X, use 'brew install gnu-sed'." >&2
  exit 1
fi

# TODO: dry this out as well
ARCHITECTURES=(
    "amd64"
    "arm64"
    "ppc64le"
    "s390x"
)

CONTAINERD_BASE_URL="https://github.com/kind-ci/containerd-nightlies/releases/download/containerd-${CONTAINERD_VERSION}"
for ARCH in "${ARCHITECTURES[@]}"; do
    CONTAINERD_URL="${CONTAINERD_BASE_URL}/containerd-${CONTAINERD_VERSION}-linux-${ARCH}.tar.gz.sha256sum"
    SHASUM=$(curl -sSL --retry 5 "${CONTAINERD_URL}" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG CONTAINERD_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
    $SED -i 's/ARG CONTAINERD_'"${ARCH_UPPER}"'_SHA256SUM=.*/ARG CONTAINERD_'"${ARCH_UPPER}"'_SHA256SUM="'"${SHASUM}"'"/' ./images/base/Dockerfile
done

echo
for ARCH in "${ARCHITECTURES[@]}"; do
    RUNC_URL="${CONTAINERD_BASE_URL}/runc.${ARCH}.sha256sum"
    SHASUM=$(curl -sSL --retry 5 "${RUNC_URL}" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG RUNC_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
    $SED -i 's/ARG RUNC_'"${ARCH_UPPER}"'_SHA256SUM=.*/ARG RUNC_'"${ARCH_UPPER}"'_SHA256SUM="'"${SHASUM}"'"/' ./images/base/Dockerfile
done

echo
for ARCH in "${ARCHITECTURES[@]}"; do
    CRICTL_URL="https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-${ARCH}.tar.gz"
    SHASUM=$(curl -sSL --retry 5 "${CRICTL_URL}.sha256" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG CRICTL_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
    $SED -i 's/ARG CRICTL_'"${ARCH_UPPER}"'_SHA256SUM=.*/ARG CRICTL_'"${ARCH_UPPER}"'_SHA256SUM="'"${SHASUM}"'"/' ./images/base/Dockerfile
done

echo
for ARCH in "${ARCHITECTURES[@]}"; do
    CNI_TARBALL="${CNI_PLUGINS_VERSION}/cni-plugins-linux-${ARCH}-${CNI_PLUGINS_VERSION}.tgz"
    CNI_URL="https://github.com/containernetworking/plugins/releases/download/${CNI_TARBALL}"
    SHASUM=$(curl -sSL --retry 5 "${CNI_URL}.sha256" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG CNI_PLUGINS_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
    $SED -i 's/ARG CNI_PLUGINS_'"${ARCH_UPPER}"'_SHA256SUM=.*/ARG CNI_PLUGINS_'"${ARCH_UPPER}"'_SHA256SUM="'"${SHASUM}"'"/' ./images/base/Dockerfile
done

echo
for ARCH in "${ARCHITECTURES[@]}"; do
    CONTAINERD_FUSE_OVERLAYFS_TARBALL="containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION}-linux-${ARCH}.tar.gz"
    CONTAINERD_FUSE_OVERLAYFS_URL="https://github.com/containerd/fuse-overlayfs-snapshotter/releases/download/v${CONTAINERD_FUSE_OVERLAYFS_VERSION}/SHA256SUMS"
    SHASUM=$(curl -sSL --retry 5 "${CONTAINERD_FUSE_OVERLAYFS_URL}" | grep "${CONTAINERD_FUSE_OVERLAYFS_TARBALL}" | awk '{print $1}')
    ARCH_UPPER=$(echo "$ARCH" | tr '[:lower:]' '[:upper:]')
    echo "ARG CONTAINERD_FUSE_OVERLAYFS_${ARCH_UPPER}_SHA256SUM=${SHASUM}"
    $SED -i 's/ARG CONTAINERD_FUSE_OVERLAYFS_'"${ARCH_UPPER}"'_SHA256SUM=.*/ARG CONTAINERD_FUSE_OVERLAYFS_'"${ARCH_UPPER}"'_SHA256SUM="'"${SHASUM}"'"/' ./images/base/Dockerfile
done
