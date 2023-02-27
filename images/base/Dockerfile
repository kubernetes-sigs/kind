# Copyright 2018 The Kubernetes Authors.
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

# kind node base image
#
# For systemd + docker configuration used below, see the following references:
# https://systemd.io/CONTAINER_INTERFACE/

# start from ubuntu, this image is reasonably small as a starting point
# for a kubernetes node image, it doesn't contain much we don't need
ARG BASE_IMAGE=ubuntu:22.04
FROM $BASE_IMAGE as build

# `docker buildx` automatically sets this arg value
ARG TARGETARCH

# Configure containerd and runc binaries from kind-ci/containerd-nightlies repository
# The repository contains latest stable releases and nightlies built for multiple architectures
ARG CONTAINERD_VERSION="1.6.18"
ARG CONTAINERD_BASE_URL="https://github.com/kind-ci/containerd-nightlies/releases/download"
ARG CONTAINERD_URL="${CONTAINERD_BASE_URL}/containerd-${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CONTAINERD_AMD64_SHA256SUM="546e315e35bbf5599fc66356d80fee20fb853967bedd9119cd6046c85a49384f"
ARG CONTAINERD_ARM64_SHA256SUM="7759c8369e2bb5296bf54e1d081ce0870015631fbe0c70a3c6db971557b6cbcb"
ARG CONTAINERD_PPC64LE_SHA256SUM="3a84530fbfcd010252b01adcc957a67b0650014ed539845bbbd7b562b1a2b575"
ARG CONTAINERD_S390X_SHA256SUM="d45544b24c4e046ef098f7dc3107a375ed1bedaee66da721f8301f528028af0b"

ARG RUNC_URL="${CONTAINERD_BASE_URL}/containerd-${CONTAINERD_VERSION}/runc.${TARGETARCH}"
ARG RUNC_AMD64_SHA256SUM="25a225cbf37df7bfc6b1f95b01508a3b82631d8edbdb792be177ebd7d65d6303"
ARG RUNC_ARM64_SHA256SUM="352ba6ca42475194265017a5c35a9bc62968922c92e2de9d2f1a8fd8112590dd"
ARG RUNC_PPC64LE_SHA256SUM="42436cb2404625f50f12fc66cc984760a75f22159ef8b38d59d792a781984e60"
ARG RUNC_S390X_SHA256SUM="b5900842e37df6120d1cae476c0ae349c0cd27f0036defbaada7a26bf4fb24b2"

# Configure crictl binary from upstream
ARG CRICTL_VERSION="v1.26.0"
ARG CRICTL_URL="https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CRICTL_AMD64_SHA256SUM="cda5e2143bf19f6b548110ffba0fe3565e03e8743fadd625fee3d62fc4134eed"
ARG CRICTL_ARM64_SHA256SUM="b632ca705a98edc8ad7806f4279feaff956ac83aa109bba8a85ed81e6b900599"
ARG CRICTL_PPC64LE_SHA256SUM="5538c88b8ccde419e6158ab9c06dfcca1fa0abecf33d0a75b2d22ceddd283f0d"
ARG CRICTL_S390X_SHA256SUM="049390a562ee4ee62ab015d4afd2616126b6a60ea640e33ae9360a080e4c2243"

# Configure CNI binaries from upstream
ARG CNI_PLUGINS_VERSION="v1.2.0"
ARG CNI_PLUGINS_TARBALL="${CNI_PLUGINS_VERSION}/cni-plugins-linux-${TARGETARCH}-${CNI_PLUGINS_VERSION}.tgz"
ARG CNI_PLUGINS_URL="https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_TARBALL}"
ARG CNI_PLUGINS_AMD64_SHA256SUM="f3a841324845ca6bf0d4091b4fc7f97e18a623172158b72fc3fdcdb9d42d2d37"
ARG CNI_PLUGINS_ARM64_SHA256SUM="525e2b62ba92a1b6f3dc9612449a84aa61652e680f7ebf4eff579795fe464b57"
ARG CNI_PLUGINS_PPC64LE_SHA256SUM="4960283b88d53b8c45ff7a938a6b398724005313e0388e0a36bd6d0b2bb5acdc"
ARG CNI_PLUGINS_S390X_SHA256SUM="1524d1e6cc237ef756040ec1b4c397659bc14df25865bfcc5ea647357ef974f2"

# Configure containerd-fuse-overlayfs snapshotter binary from upstream
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION="1.0.5"
ARG CONTAINERD_FUSE_OVERLAYFS_TARBALL="v${CONTAINERD_FUSE_OVERLAYFS_VERSION}/containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CONTAINERD_FUSE_OVERLAYFS_URL="https://github.com/containerd/fuse-overlayfs-snapshotter/releases/download/${CONTAINERD_FUSE_OVERLAYFS_TARBALL}"
ARG CONTAINERD_FUSE_OVERLAYFS_AMD64_SHA256SUM="1f4b12322cc1b044dfbbeaec30fc42295cedc8b6f0642146ba518333f9d5ddca"
ARG CONTAINERD_FUSE_OVERLAYFS_ARM64_SHA256SUM="073e83196a7a73bd130fe44085bd65303c7e6cfc8c53ba46d90a16cbb8e5a112"
ARG CONTAINERD_FUSE_OVERLAYFS_PPC64LE_SHA256SUM="1a3914ce6780be65a901ceb10f6ed49ecdc3754f51b9988119fcf64840d67f60"
ARG CONTAINERD_FUSE_OVERLAYFS_S390X_SHA256SUM="eb3c7d11c5440bada47be635dd08a2809858e0225c27c98e421ccd342ccbc60b"

# copy in static files
# all scripts are 0755 (rwx r-x r-x)
COPY --chmod=0755 files/usr/local/bin/* /usr/local/bin/

# all configs are 0644 (rw- r-- r--)
COPY --chmod=0644 files/etc/* /etc/
COPY --chmod=0644 files/etc/containerd/* /etc/containerd/
COPY --chmod=0644 files/etc/default/* /etc/default/
COPY --chmod=0644 files/etc/sysctl.d/* /etc/sysctl.d/
COPY --chmod=0644 files/etc/systemd/system/* /etc/systemd/system/
COPY --chmod=0644 files/etc/systemd/system/kubelet.service.d/* /etc/systemd/system/kubelet.service.d/

# Install dependencies, first from apt, then from release tarballs.
# NOTE: we use one RUN to minimize layers.
#
# First we must ensure that our util scripts are executable.
#
# The base image already has a basic userspace + apt but we need to install more packages.
# Packages installed are broken down into (each on a line):
# - packages needed to run services (systemd)
# - packages needed for kubernetes components
# - packages needed by the container runtime
# - misc packages kind uses itself
# - packages that provide semi-core kubernetes functionality
# After installing packages we cleanup by:
# - removing unwanted systemd services
# - disabling kmsg in journald (these log entries would be confusing)
#
# Then we install containerd from our nightly build infrastructure, as this
# build for multiple architectures and allows us to upgrade to patched releases
# more quickly.
#
# Next we download and extract crictl and CNI plugin binaries from upstream.
#
# Next we ensure the /etc/kubernetes/manifests directory exists. Normally
# a kubeadm debian / rpm package would ensure that this exists but we install
# freshly built binaries directly when we build the node image.
#
# Finally we adjust tempfiles cleanup to be 1 minute after "boot" instead of 15m
# This is plenty after we've done initial setup for a node, but before we are
# likely to try to export logs etc.

RUN echo "Installing Packages ..." \
    && DEBIAN_FRONTEND=noninteractive clean-install \
      systemd \
      conntrack iptables iproute2 ethtool socat util-linux mount ebtables kmod \
      libseccomp2 pigz \
      bash ca-certificates curl rsync \
      nfs-common fuse-overlayfs open-iscsi \
      jq \
    && find /lib/systemd/system/sysinit.target.wants/ -name "systemd-tmpfiles-setup.service" -delete \
    && rm -f /lib/systemd/system/multi-user.target.wants/* \
    && rm -f /etc/systemd/system/*.wants/* \
    && rm -f /lib/systemd/system/local-fs.target.wants/* \
    && rm -f /lib/systemd/system/sockets.target.wants/*udev* \
    && rm -f /lib/systemd/system/sockets.target.wants/*initctl* \
    && rm -f /lib/systemd/system/basic.target.wants/* \
    && echo "ReadKMsg=no" >> /etc/systemd/journald.conf \
    && ln -s "$(which systemd)" /sbin/init

RUN echo "Enabling kubelet ... " \
    && systemctl enable kubelet.service

RUN echo "Installing containerd ..." \
    && curl -sSL --retry 5 --output /tmp/containerd.${TARGETARCH}.tgz "${CONTAINERD_URL}" \
    && echo "${CONTAINERD_AMD64_SHA256SUM}  /tmp/containerd.amd64.tgz" | tee /tmp/containerd.sha256 \
    && echo "${CONTAINERD_ARM64_SHA256SUM}  /tmp/containerd.arm64.tgz" | tee -a /tmp/containerd.sha256 \
    && echo "${CONTAINERD_PPC64LE_SHA256SUM}  /tmp/containerd.ppc64le.tgz" | tee -a /tmp/containerd.sha256 \
    && echo "${CONTAINERD_S390X_SHA256SUM}  /tmp/containerd.s390x.tgz" | tee -a /tmp/containerd.sha256 \
    && sha256sum --ignore-missing -c /tmp/containerd.sha256 \
    && rm -f /tmp/containerd.sha256 \
    && tar -C /usr/local -xzvf /tmp/containerd.${TARGETARCH}.tgz \
    && rm -rf /tmp/containerd.${TARGETARCH}.tgz \
    && rm -f /usr/local/bin/containerd-stress /usr/local/bin/containerd-shim-runc-v1 \
    && curl -sSL --retry 5 --output /tmp/runc.${TARGETARCH} "${RUNC_URL}" \
    && echo "${RUNC_AMD64_SHA256SUM}  /tmp/runc.amd64" | tee /tmp/runc.sha256 \
    && echo "${RUNC_ARM64_SHA256SUM}  /tmp/runc.arm64" | tee -a /tmp/runc.sha256 \
    && echo "${RUNC_PPC64LE_SHA256SUM}  /tmp/runc.ppc64le" | tee -a /tmp/runc.sha256 \
    && echo "${RUNC_S390X_SHA256SUM}  /tmp/runc.s390x" | tee -a /tmp/runc.sha256 \
    && sha256sum --ignore-missing -c /tmp/runc.sha256 \
    && mv /tmp/runc.${TARGETARCH} /usr/local/sbin/runc \
    && chmod 755 /usr/local/sbin/runc \
    && ctr oci spec \
        | jq '.hooks.createContainer[.hooks.createContainer| length] |= . + {"path": "/usr/local/bin/mount-product-files"}' \
        | jq 'del(.process.rlimits)' \
        > /etc/containerd/cri-base.json \
    && containerd --version \
    && runc --version \
    && systemctl enable containerd

RUN echo "Installing crictl ..." \
    && curl -sSL --retry 5 --output /tmp/crictl.${TARGETARCH}.tgz "${CRICTL_URL}" \
    && echo "${CRICTL_AMD64_SHA256SUM}  /tmp/crictl.amd64.tgz" | tee /tmp/crictl.sha256 \
    && echo "${CRICTL_ARM64_SHA256SUM}  /tmp/crictl.arm64.tgz" | tee -a /tmp/crictl.sha256 \
    && echo "${CRICTL_PPC64LE_SHA256SUM}  /tmp/crictl.ppc64le.tgz" | tee -a /tmp/crictl.sha256 \
    && echo "${CRICTL_S390X_SHA256SUM}  /tmp/crictl.s390x.tgz" | tee -a /tmp/crictl.sha256 \
    && sha256sum --ignore-missing -c /tmp/crictl.sha256 \
    && rm -f /tmp/crictl.sha256 \
    && tar -C /usr/local/bin -xzvf /tmp/crictl.${TARGETARCH}.tgz \
    && rm -rf /tmp/crictl.${TARGETARCH}.tgz

RUN echo "Installing CNI plugin binaries ..." \
    && curl -sSL --retry 5 --output /tmp/cni.${TARGETARCH}.tgz "${CNI_PLUGINS_URL}" \
    && echo "${CNI_PLUGINS_AMD64_SHA256SUM}  /tmp/cni.amd64.tgz" | tee /tmp/cni.sha256 \
    && echo "${CNI_PLUGINS_ARM64_SHA256SUM}  /tmp/cni.arm64.tgz" | tee -a /tmp/cni.sha256 \
    && echo "${CNI_PLUGINS_PPC64LE_SHA256SUM}  /tmp/cni.ppc64le.tgz" | tee -a /tmp/cni.sha256 \
    && echo "${CNI_PLUGINS_S390X_SHA256SUM}  /tmp/cni.s390x.tgz" | tee -a /tmp/cni.sha256 \
    && sha256sum --ignore-missing -c /tmp/cni.sha256 \
    && rm -f /tmp/cni.sha256 \
    && mkdir -p /opt/cni/bin \
    && tar -C /opt/cni/bin -xzvf /tmp/cni.${TARGETARCH}.tgz \
    && rm -rf /tmp/cni.${TARGETARCH}.tgz \
    && find /opt/cni/bin -type f -not \( \
         -iname host-local \
         -o -iname ptp \
         -o -iname portmap \
         -o -iname loopback \
      \) \
      -delete

RUN echo "Installing containerd-fuse-overlayfs ..." \
    && curl -sSL --retry 5 --output /tmp/containerd-fuse-overlayfs.${TARGETARCH}.tgz "${CONTAINERD_FUSE_OVERLAYFS_URL}" \
    && echo "${CONTAINERD_FUSE_OVERLAYFS_AMD64_SHA256SUM}  /tmp/containerd-fuse-overlayfs.amd64.tgz" | tee /tmp/containerd-fuse-overlayfs.sha256 \
    && echo "${CONTAINERD_FUSE_OVERLAYFS_ARM64_SHA256SUM}  /tmp/containerd-fuse-overlayfs.arm64.tgz" | tee -a /tmp/containerd-fuse-overlayfs.sha256 \
    && echo "${CONTAINERD_FUSE_OVERLAYFS_PPC64LE_SHA256SUM}  /tmp/containerd-fuse-overlayfs.ppc64le.tgz" | tee -a /tmp/containerd-fuse-overlayfs.sha256 \
    && echo "${CONTAINERD_FUSE_OVERLAYFS_S390X_SHA256SUM}  /tmp/containerd-fuse-overlayfs.s390x.tgz" | tee -a /tmp/containerd-fuse-overlayfs.sha256 \
    && sha256sum --ignore-missing -c /tmp/containerd-fuse-overlayfs.sha256 \
    && rm -f /tmp/containerd-fuse-overlayfs.sha256 \
    && tar -C /usr/local/bin -xzvf /tmp/containerd-fuse-overlayfs.${TARGETARCH}.tgz \
    && rm -rf /tmp/containerd-fuse-overlayfs.${TARGETARCH}.tgz

RUN echo "Ensuring /etc/kubernetes/manifests" \
    && mkdir -p /etc/kubernetes/manifests

RUN echo "Adjusting systemd-tmpfiles timer" \
    && sed -i /usr/lib/systemd/system/systemd-tmpfiles-clean.timer -e 's#OnBootSec=.*#OnBootSec=1min#'

# squash
FROM scratch
COPY --from=build / /

# tell systemd that it is in docker (it will check for the container env)
# https://systemd.io/CONTAINER_INTERFACE/
ENV container docker
# systemd exits on SIGRTMIN+3, not SIGTERM (which re-executes it)
# https://bugzilla.redhat.com/show_bug.cgi?id=1201657
STOPSIGNAL SIGRTMIN+3
# NOTE: this is *only* for documentation, the entrypoint is overridden later
ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]
