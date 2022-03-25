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
ARG BASE_IMAGE=ubuntu:21.10
FROM $BASE_IMAGE as build

# `docker buildx` automatically sets this arg value
ARG TARGETARCH

# Configure containerd and runc binaries from kind-ci/containerd-nightlies repository
# The repository contains latest stable releases and nightlies built for multiple architectures
ARG CONTAINERD_VERSION="1.6.2"
ARG CONTAINERD_BASE_URL="https://github.com/kind-ci/containerd-nightlies/releases/download/containerd-${CONTAINERD_VERSION}"
ARG CONTAINERD_URL="${CONTAINERD_BASE_URL}/containerd-${CONTAINERD_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CONTAINERD_AMD64_SHA256SUM="be9efcdefdec84023ffba91cb55617b5ac3f178b6f3b1ccce314e3e2dad74d68"
ARG CONTAINERD_ARM64_SHA256SUM="ecc43b17bffe8fafa92ebc294859c3713d058872314fc1674349eff88709b02c"
ARG CONTAINERD_PPC64LE_SHA256SUM="6ea85f09e2a01f6f01666ac3842d6a210d905e1ed01e98beb8c4c7f547ef2e4d"
ARG CONTAINERD_S390X_SHA256SUM="cc1b634dc53a30bc06eb0bb8ff82e42ff13802e682f114cf24190055e2c95be0"

ARG RUNC_URL="${CONTAINERD_BASE_URL}/runc.${TARGETARCH}"
ARG RUNC_AMD64_SHA256SUM="b99c4ed069047a7e90e3e23b1ad456a5b6d757ffebb2de1ad6c29ab66c050b82"
ARG RUNC_ARM64_SHA256SUM="c4de4327050cbe12bd53e6102426a4a0ed4ce343ac55537dc37ac044f9d7c33d"
ARG RUNC_PPC64LE_SHA256SUM="1d9bb1a547e954ebf173514b3215a4d07520eade43de3e677009116fadeea36c"
ARG RUNC_S390X_SHA256SUM="9084be4b463bcfb0908ee49678892afce6ca826bd8aca114c08bbf7041acf373"

# Configure crictl binary from upstream
ARG CRICTL_VERSION="v1.23.0"
ARG CRICTL_URL="https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CRICTL_AMD64_SHA256SUM="b754f83c80acdc75f93aba191ff269da6be45d0fc2d3f4079704e7d1424f1ca8"
ARG CRICTL_ARM64_SHA256SUM="91094253e77094435027998a99b9b6a67b0baad3327975365f7715a1a3bd9595"
ARG CRICTL_PPC64LE_SHA256SUM="53db9e605a3042ea77bbf42a01a4e248dea8839bcab544c491745874f73aeee7"
ARG CRICTL_S390X_SHA256SUM="2f9e24ec7b5aeb935f735a387257b17c62f4fb6c9cb0448d4f929a28eed0710a"

# Configure CNI binaries from upstream
ARG CNI_PLUGINS_VERSION="v1.1.1"
ARG CNI_PLUGINS_TARBALL="${CNI_PLUGINS_VERSION}/cni-plugins-linux-${TARGETARCH}-${CNI_PLUGINS_VERSION}.tgz"
ARG CNI_PLUGINS_URL="https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_TARBALL}"
ARG CNI_PLUGINS_AMD64_SHA256SUM="b275772da4026d2161bf8a8b41ed4786754c8a93ebfb6564006d5da7f23831e5"
ARG CNI_PLUGINS_ARM64_SHA256SUM="16484966a46b4692028ba32d16afd994e079dc2cc63fbc2191d7bfaf5e11f3dd"
ARG CNI_PLUGINS_PPC64LE_SHA256SUM="1551259fbfe861d942846bee028d5a85f492393e04bcd6609ac8aaa7a3d71431"
ARG CNI_PLUGINS_S390X_SHA256SUM="767c6b2f191a666522ab18c26aab07de68508a8c7a6d56625e476f35ba527c76"

# Configure containerd-fuse-overlayfs snapshotter binary from upstream
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION="1.0.4"
ARG CONTAINERD_FUSE_OVERLAYFS_TARBALL="v${CONTAINERD_FUSE_OVERLAYFS_VERSION}/containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CONTAINERD_FUSE_OVERLAYFS_URL="https://github.com/containerd/fuse-overlayfs-snapshotter/releases/download/${CONTAINERD_FUSE_OVERLAYFS_TARBALL}"
ARG CONTAINERD_FUSE_OVERLAYFS_AMD64_SHA256SUM="228417cc97fea4df26ed85182443ee4d5799f65ada0b3ce663bf7e6bc8920f6b"
ARG CONTAINERD_FUSE_OVERLAYFS_ARM64_SHA256SUM="5ede755ff8fe2cb3e38b59d7eb005ccc29a88037a2fa5f7749d28471ee81c727"
ARG CONTAINERD_FUSE_OVERLAYFS_PPC64LE_SHA256SUM="cd4775b6de118bc3bc83efb89e4303c96c602b86d63f2df4a083f48e968c171d"
ARG CONTAINERD_FUSE_OVERLAYFS_S390X_SHA256SUM="55dae4a2c74b1215ec99591b9197e6a651bfcc6f96b4cc0331537fba5411eadc"

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
      nfs-common fuse-overlayfs \
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
