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

# start from ubuntu 20.10, this image is reasonably small as a starting point
# for a kubernetes node image, it doesn't contain much we don't need
ARG BASE_IMAGE=ubuntu:20.10
FROM $BASE_IMAGE

# `docker buildx` automatically sets this arg value, but we add the arg for
# regular `docker bulid` invocations to force a selection
ARG TARGETARCH

# Configure containerd and runc binaries from kind-ci/containerd-nightlies repository
# The repository contains latest stable releases and nightlies built for multiple architectures
ARG CONTAINERD_VERSION="1.5.0-beta.1-28-g9173d3e92"
ARG CONTAINERD_BASE_URL="https://github.com/kind-ci/containerd-nightlies/releases/download/containerd-${CONTAINERD_VERSION}"
ARG CONTAINERD_URL="${CONTAINERD_BASE_URL}/containerd-${CONTAINERD_VERSION}.linux-${TARGETARCH}.tar.gz"
ARG CONTAINERD_AMD64_SHA256SUM="f73e5895d82f8bb74bddb614a399f4d7a57f3e0b62f0938e1ca9692aa2e84b61"
ARG CONTAINERD_ARM64_SHA256SUM="1302ccde89e652cbacb34b9bc54964e5dae05eebfd8c46cd6bb67e6faaf88c99"
ARG CONTAINERD_PPC64LE_SHA256SUM="0a2260fd96fc6aca306b3bf40472a4b39fd552acf6b36414d280c1ebfa480417"

ARG RUNC_URL="${CONTAINERD_BASE_URL}/runc.${TARGETARCH}"
ARG RUNC_AMD64_SHA256SUM="28b0981b86050df6f2a7ccbc573c5f2ed74b7da7da31589d55c45119b0fc7f73"
ARG RUNC_ARM64_SHA256SUM="287e06abe20fb882f68ba07c0bf953f94ea5a6c4617dc0b00740b6606efdf233"
ARG RUNC_PPC64LE_SHA256SUM="9c8563cd3660a480264ecc345e63d67e33e0c38da00341b83316b9a67a565645"

# Configure crictl binary from upstream
ARG CRICTL_VERSION="v1.20.0"
ARG CRICTL_URL="https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CRICTL_AMD64_SHA256SUM="44d5f550ef3f41f9b53155906e0229ffdbee4b19452b4df540265e29572b899c"
ARG CRICTL_ARM64_SHA256SUM="eda6879710eb046d335162d4afe8494c6f8161142ad3188022852f64b92806a8"
ARG CRICTL_PPC64LE_SHA256SUM="da0c052983ba884f9605b14bf627664df67fcdb41c7f6908368bf4745f889b26"

# Configure CNI binaries from upstream
ARG CNI_PLUGINS_VERSION="v0.9.0"
ARG CNI_PLUGINS_TARBALL="${CNI_PLUGINS_VERSION}/cni-plugins-linux-${TARGETARCH}-${CNI_PLUGINS_VERSION}.tgz"
ARG CNI_PLUGINS_URL="https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_TARBALL}"
ARG CNI_PLUGINS_AMD64_SHA256SUM="58a58d389895ba9f9bbd3ef330f186c0bb7484136d0bfb9b50152eed55d9ec24"
ARG CNI_PLUGINS_ARM64_SHA256SUM="49bdf1d3c852a831964aea8c9d12340b36107ee756d8328403905ff599abc6f5"
ARG CNI_PLUGINS_PPC64LE_SHA256SUM="d37829b5eeca0c941b4478203c75c6cc26d9cfc1d6c8bb451c0008e0c02a025f"

# copy in static files (configs, scripts)
COPY files/ /

# Install dependencies, first from apt, then from release tarballs.
# NOTE: we use one RUN to minimize layers.
#
# First we must ensure that our util scripts are executable.
#
# The base image already has: ssh, apt, snapd, but we need to install more packages.
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
RUN echo "Ensuring scripts are executable ..." \
    && chmod +x /usr/local/bin/clean-install /usr/local/bin/entrypoint \
 && echo "Installing Packages ..." \
    && DEBIAN_FRONTEND=noninteractive clean-install \
      systemd \
      conntrack iptables iproute2 ethtool socat util-linux mount ebtables udev kmod \
      libseccomp2 pigz \
      bash ca-certificates curl rsync \
      nfs-common \
    && find /lib/systemd/system/sysinit.target.wants/ -name "systemd-tmpfiles-setup.service" -delete \
    && rm -f /lib/systemd/system/multi-user.target.wants/* \
    && rm -f /etc/systemd/system/*.wants/* \
    && rm -f /lib/systemd/system/local-fs.target.wants/* \
    && rm -f /lib/systemd/system/sockets.target.wants/*udev* \
    && rm -f /lib/systemd/system/sockets.target.wants/*initctl* \
    && rm -f /lib/systemd/system/basic.target.wants/* \
    && echo "ReadKMsg=no" >> /etc/systemd/journald.conf \
    && ln -s "$(which systemd)" /sbin/init \
 && echo "Enabling kubelet ... " \
    && systemctl enable kubelet.service \
 && echo "Installing containerd ..." \
    && curl -sSL --retry 5 --output /tmp/containerd.${TARGETARCH}.tgz "${CONTAINERD_URL}" \
    && echo "${CONTAINERD_AMD64_SHA256SUM}  /tmp/containerd.amd64.tgz" | tee /tmp/containerd.sha256 \
    && echo "${CONTAINERD_ARM64_SHA256SUM}  /tmp/containerd.arm64.tgz" | tee -a /tmp/containerd.sha256 \
    && echo "${CONTAINERD_PPC64LE_SHA256SUM}  /tmp/containerd.ppc64le.tgz" | tee -a /tmp/containerd.sha256 \
    && sha256sum --ignore-missing -c /tmp/containerd.sha256 \
    && rm -f /tmp/containerd.sha256 \
    && tar -C /usr/local -xzvf /tmp/containerd.${TARGETARCH}.tgz \
    && rm -rf /tmp/containerd.${TARGETARCH}.tgz \
    && rm -f /usr/local/bin/containerd-stress /usr/local/bin/containerd-shim-runc-v1 \
    && curl -sSL --retry 5 --output /tmp/runc.${TARGETARCH} "${RUNC_URL}" \
    && echo "${RUNC_AMD64_SHA256SUM}  /tmp/runc.amd64" | tee /tmp/runc.sha256 \
    && echo "${RUNC_ARM64_SHA256SUM}  /tmp/runc.arm64" | tee -a /tmp/runc.sha256 \
    && echo "${RUNC_PPC64LE_SHA256SUM}  /tmp/runc.ppc64le" | tee -a /tmp/runc.sha256 \
    && sha256sum --ignore-missing -c /tmp/runc.sha256 \
    && mv /tmp/runc.${TARGETARCH} /usr/local/sbin/runc \
    && chmod 755 /usr/local/sbin/runc \
    && containerd --version \
    && runc --version \
    && systemctl enable containerd \
 && echo "Installing crictl ..." \
    && curl -sSL --retry 5 --output /tmp/crictl.${TARGETARCH}.tgz "${CRICTL_URL}" \
    && echo "${CRICTL_AMD64_SHA256SUM}  /tmp/crictl.amd64.tgz" | tee /tmp/crictl.sha256 \
    && echo "${CRICTL_ARM64_SHA256SUM}  /tmp/crictl.arm64.tgz" | tee -a /tmp/crictl.sha256 \
    && echo "${CRICTL_PPC64LE_SHA256SUM}  /tmp/crictl.ppc64le.tgz" | tee -a /tmp/crictl.sha256 \
    && sha256sum --ignore-missing -c /tmp/crictl.sha256 \
    && rm -f /tmp/crictl.sha256 \
    && tar -C /usr/local/bin -xzvf /tmp/crictl.${TARGETARCH}.tgz \
    && rm -rf /tmp/crictl.${TARGETARCH}.tgz \
 && echo "Installing CNI plugin binaries ..." \
    && curl -sSL --retry 5 --output /tmp/cni.${TARGETARCH}.tgz "${CNI_PLUGINS_URL}" \
    && echo "${CNI_PLUGINS_AMD64_SHA256SUM}  /tmp/cni.amd64.tgz" | tee /tmp/cni.sha256 \
    && echo "${CNI_PLUGINS_ARM64_SHA256SUM}  /tmp/cni.arm64.tgz" | tee -a /tmp/cni.sha256 \
    && echo "${CNI_PLUGINS_PPC64LE_SHA256SUM}  /tmp/cni.ppc64le.tgz" | tee -a /tmp/cni.sha256 \
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
      -delete \
 && echo "Ensuring /etc/kubernetes/manifests" \
    && mkdir -p /etc/kubernetes/manifests \
 && echo "Adjusting systemd-tmpfiles timer" \
    && sed -i /usr/lib/systemd/system/systemd-tmpfiles-clean.timer -e 's#OnBootSec=.*#OnBootSec=1min#' \
 && echo "Disabling udev" \
    && systemctl disable udev.service

# tell systemd that it is in docker (it will check for the container env)
# https://systemd.io/CONTAINER_INTERFACE/
ENV container docker
# systemd exits on SIGRTMIN+3, not SIGTERM (which re-executes it)
# https://bugzilla.redhat.com/show_bug.cgi?id=1201657
STOPSIGNAL SIGRTMIN+3
# NOTE: this is *only* for documentation, the entrypoint is overridden later
ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]
