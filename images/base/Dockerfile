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
ARG CONTAINERD_VERSION="1.5.9"
ARG CONTAINERD_BASE_URL="https://github.com/kind-ci/containerd-nightlies/releases/download/containerd-${CONTAINERD_VERSION}"
ARG CONTAINERD_URL="${CONTAINERD_BASE_URL}/containerd-${CONTAINERD_VERSION}.linux-${TARGETARCH}.tar.gz"
ARG CONTAINERD_AMD64_SHA256SUM="51a2fdf8cc4f50bcd4062e2142d5ac248817e47db1406c761aa478748b27f696"
ARG CONTAINERD_ARM64_SHA256SUM="363cc9766ed90266092799600d214d4f3c1eefa197799e42076311dc24955448"
ARG CONTAINERD_PPC64LE_SHA256SUM="37207b653397789cc16492d5362aa7c052700c3dc9423552e07dd3b199aa1994"
ARG CONTAINERD_S390X_SHA256SUM="ec73c6aa16e8bf213fdea6f413fbaf63cc4447915b2dce5d6cc1779bf51c9f0b"

ARG RUNC_URL="${CONTAINERD_BASE_URL}/runc.${TARGETARCH}"
ARG RUNC_AMD64_SHA256SUM="8519c8ff8afcd9a9990d3e833a693dad999f02d88d258004236f3d7580b09bb7"
ARG RUNC_ARM64_SHA256SUM="5662a389501c435a0e5842771a3a63972f4a0f833f05c96de5ca13b9da4a1dba"
ARG RUNC_PPC64LE_SHA256SUM="940f0f1839b3fb5ca5aab5cfb718166a2a7fc39c0edb53ac7dfa108bc08baa3c"
ARG RUNC_S390X_SHA256SUM="c55239f15a4062dd63584d552e56dee638d6b68df98dc849b455a6e5dd77c7fd"

# Configure crictl binary from upstream
ARG CRICTL_VERSION="v1.22.0"
ARG CRICTL_URL="https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CRICTL_AMD64_SHA256SUM="45e0556c42616af60ebe93bf4691056338b3ea0001c0201a6a8ff8b1dbc0652a"
ARG CRICTL_ARM64_SHA256SUM="a713c37fade0d96a989bc15ebe906e08ef5c8fe5e107c2161b0665e9963b770e"
ARG CRICTL_PPC64LE_SHA256SUM="c78bcea20c8f8ca3be0762cca7349fd2f1df520c304d0b2ef5e8fa514f64e45f"
ARG CRICTL_S390X_SHA256SUM="2afcf677b1c5665d0cd0f751fd5b5d7c1db6f063e007aa6b897bb5ac319611d9"

# Configure CNI binaries from upstream
ARG CNI_PLUGINS_VERSION="v1.0.1"
ARG CNI_PLUGINS_TARBALL="${CNI_PLUGINS_VERSION}/cni-plugins-linux-${TARGETARCH}-${CNI_PLUGINS_VERSION}.tgz"
ARG CNI_PLUGINS_URL="https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_TARBALL}"
ARG CNI_PLUGINS_AMD64_SHA256SUM="5238fbb2767cbf6aae736ad97a7aa29167525dcd405196dfbc064672a730d3cf"
ARG CNI_PLUGINS_ARM64_SHA256SUM="2d4528c45bdd0a8875f849a75082bc4eafe95cb61f9bcc10a6db38a031f67226"
ARG CNI_PLUGINS_PPC64LE_SHA256SUM="f078e33067e6daaef3a3a5010d6440f2464b7973dec3ca0b5d5be22fdcb1fd96"
ARG CNI_PLUGINS_S390X_SHA256SUM="468d33e16440d9ca4395c6bb2d5b71b35ae4a4df26301e4da85ac70c5ce56822"

# Configure containerd-fuse-overlayfs snapshotter binary from upstream
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION="1.0.3"
ARG CONTAINERD_FUSE_OVERLAYFS_TARBALL="v${CONTAINERD_FUSE_OVERLAYFS_VERSION}/containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION}-linux-${TARGETARCH}.tar.gz"
ARG CONTAINERD_FUSE_OVERLAYFS_URL="https://github.com/containerd/fuse-overlayfs-snapshotter/releases/download/${CONTAINERD_FUSE_OVERLAYFS_TARBALL}"
ARG CONTAINERD_FUSE_OVERLAYFS_AMD64_SHA256SUM="26c7af08d292f21e7067c0424479945bb9ff6315b49851511b2917179c5ae59a"
ARG CONTAINERD_FUSE_OVERLAYFS_ARM64_SHA256SUM="68ef0896f3d5c0af73ad3d13b1b9a27f9b57cf22bdc30e36915d0f279b965bc3"
ARG CONTAINERD_FUSE_OVERLAYFS_PPC64LE_SHA256SUM="49679827fa2b46dd28899bdc53c2926e83f42d305ad7ee31aeaf50dbb774a840"
ARG CONTAINERD_FUSE_OVERLAYFS_S390X_SHA256SUM="ed74e26de3215a62154b47be67953a25a15e02f7a8550408fec541d6799bc7ad"

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
