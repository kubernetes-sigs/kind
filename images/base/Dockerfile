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

# start from debian slim, this image is reasonably small as a starting point
# for a kubernetes node image, it doesn't contain much (anything?) we don't need
# this stage will install basic files and packages
ARG BASE_IMAGE=debian:bookworm-slim
FROM $BASE_IMAGE as base

# copy in static files
# all scripts are 0755 (rwx r-x r-x)
COPY --chmod=0755 files/usr/local/bin/* /usr/local/bin/
COPY --chmod=0755 files/kind/bin/* /kind/bin/

# all non-scripts are 0644 (rw- r-- r--)
COPY --chmod=0644 files/kind/* /kind/
COPY --chmod=0644 files/LICENSES/* /LICENSES/*
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
# - packages needed for networked backed storage with kubernetes
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
      conntrack iptables iproute2 ethtool util-linux mount ebtables kmod \
      libseccomp2 pigz fuse-overlayfs \
      nfs-common open-iscsi \
      bash ca-certificates curl jq procps \
    && find /lib/systemd/system/sysinit.target.wants/ -name "systemd-tmpfiles-setup.service" -delete \
    && rm -f /lib/systemd/system/multi-user.target.wants/* \
    && rm -f /etc/systemd/system/*.wants/* \
    && rm -f /lib/systemd/system/local-fs.target.wants/* \
    && rm -f /lib/systemd/system/sockets.target.wants/*udev* \
    && rm -f /lib/systemd/system/sockets.target.wants/*initctl* \
    && rm -f /lib/systemd/system/basic.target.wants/* \
    && echo "ReadKMsg=no" >> /etc/systemd/journald.conf \
    && ln -s "$(which systemd)" /sbin/init

RUN echo "Enabling services ... " \
    && systemctl enable kubelet.service \
    && systemctl enable containerd.service \
    && systemctl enable undo-mount-hacks.service

RUN echo "Ensuring /etc/kubernetes/manifests" \
    && mkdir -p /etc/kubernetes/manifests

# shared stage to setup go version for building binaries
# NOTE we will be cross-compiling for performance reasons
# This is also why we start again FROM the same base image but a different
# platform and only the files needed for building
# We will copy the built binaries from later stages to the final stage(s)
FROM --platform=$BUILDPLATFORM $BASE_IMAGE as go-build
COPY --chmod=0755 files/usr/local/bin/* /usr/local/bin/
COPY --chmod=0755 scripts/third_party/gimme/gimme /usr/local/bin/
COPY --chmod=0755 scripts/target-cc /usr/local/bin/
# tools needed at build-time only
# first ensure we can install packages for both architectures
RUN dpkg --add-architecture arm64 && dpkg --add-architecture amd64 \
    && clean-install bash ca-certificates curl git make pkg-config \
    crossbuild-essential-amd64 crossbuild-essential-arm64 \
    libseccomp-dev:amd64 libseccomp-dev:arm64
# set by makefile to .go-version
ARG GO_VERSION
RUN eval "$(gimme "${GO_VERSION}")" \
    && GOBIN=/usr/local/bin go install github.com/google/go-licenses@latest


# stage for building containerd
FROM go-build as build-containerd
ARG TARGETARCH GO_VERSION
ARG CONTAINERD_VERSION="v1.7.5"
ARG CONTAINERD_CLONE_URL="https://github.com/containerd/containerd"
# we don't build with optional snapshotters, we never select any of these
# they're not ideal inside kind anyhow, and we save some disk space
ARG BUILDTAGS="no_aufs no_zfs no_btrfs no_devmapper"
RUN git clone --filter=tree:0 "${CONTAINERD_CLONE_URL}" /containerd \
    && cd /containerd \
    && git checkout "${CONTAINERD_VERSION}" \
    && eval "$(gimme "${GO_VERSION}")" \
    && export GOARCH=$TARGETARCH && export CC=$(target-cc) && export CGO_ENABLED=1 \
    && make bin/ctr bin/containerd bin/containerd-shim-runc-v2 \
    && GOARCH=$TARGETARCH go-licenses save --save_path=/_LICENSES \
        ./cmd/ctr ./cmd/containerd ./cmd/containerd-shim-runc-v2

# stage for building runc
FROM go-build as build-runc
ARG TARGETARCH GO_VERSION
ARG RUNC_VERSION="v1.1.9"
ARG RUNC_CLONE_URL="https://github.com/opencontainers/runc"
RUN git clone --filter=tree:0 "${RUNC_CLONE_URL}" /runc \
    && cd /runc \
    && git checkout "${RUNC_VERSION}" \
    && eval "$(gimme "${GO_VERSION}")" \
    && export GOARCH=$TARGETARCH && export CC=$(target-cc) && export CGO_ENABLED=1 \
    && make runc \
    && GOARCH=$TARGETARCH go-licenses save --save_path=/_LICENSES .

# stage for building crictl
FROM go-build as build-crictl
ARG TARGETARCH GO_VERSION
ARG CRI_TOOLS_CLONE_URL="https://github.com/kubernetes-sigs/cri-tools"
ARG CRICTL_VERSION="v1.28.0"
RUN git clone --filter=tree:0 "${CRI_TOOLS_CLONE_URL}" /cri-tools \
    && cd /cri-tools \
    && git checkout "${CRICTL_VERSION}" \
    && eval "$(gimme "${GO_VERSION}")" \
    && export GOARCH=$TARGETARCH && export CC=$(target-cc) && export CGO_ENABLED=1 \
    && make BUILD_BIN_PATH=./build crictl \
    && GOARCH=$TARGETARCH go-licenses save --save_path=/_LICENSES ./cmd/crictl

# stage for building cni-plugins
FROM go-build as build-cni
ARG TARGETARCH GO_VERSION
ARG CNI_PLUGINS_VERSION="v1.3.0"
ARG CNI_PLUGINS_CLONE_URL="https://github.com/containernetworking/plugins"
RUN git clone --filter=tree:0 "${CNI_PLUGINS_CLONE_URL}" /cni-plugins \
    && cd /cni-plugins \
    && git checkout "${CNI_PLUGINS_VERSION}" \
    && eval "$(gimme "${GO_VERSION}")" \
    && mkdir ./bin \
    && export GOARCH=$TARGETARCH && export CC=$(target-cc) && export CGO_ENABLED=1 \
    && go build -o ./bin/host-local -mod=vendor ./plugins/ipam/host-local \
    && go build -o ./bin/loopback -mod=vendor ./plugins/main/loopback \
    && go build -o ./bin/ptp -mod=vendor ./plugins/main/ptp \
    && go build -o ./bin/portmap -mod=vendor ./plugins/meta/portmap \
    && GOARCH=$TARGETARCH go-licenses save --save_path=/_LICENSES \
        ./plugins/ipam/host-local \
        ./plugins/main/loopback ./plugins/main/ptp \
        ./plugins/meta/portmap

# stage for building containerd-fuse-overlayfs
FROM go-build as build-fuse-overlayfs
ARG TARGETARCH GO_VERSION
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION="v1.0.6"
ARG CONTAINERD_FUSE_OVERLAYFS_CLONE_URL="https://github.com/containerd/fuse-overlayfs-snapshotter"
RUN git clone --filter=tree:0 "${CONTAINERD_FUSE_OVERLAYFS_CLONE_URL}" /fuse-overlayfs-snapshotter \
    && cd /fuse-overlayfs-snapshotter \
    && git checkout "${CONTAINERD_FUSE_OVERLAYFS_VERSION}" \
    && eval "$(gimme "${GO_VERSION}")" \
    && export GOARCH=$TARGETARCH && export CC=$(target-cc) && export CGO_ENABLED=1 \
    && make bin/containerd-fuse-overlayfs-grpc \
    && GOARCH=$TARGETARCH go-licenses save --save_path=/_LICENSES ./cmd/containerd-fuse-overlayfs-grpc


# build final image layout from other stages
FROM base as build
# copy over containerd build and install
COPY --from=build-containerd /containerd/bin/containerd /usr/local/bin/
COPY --from=build-containerd /containerd/bin/ctr /usr/local/bin/
COPY --from=build-containerd /containerd/bin/containerd-shim-runc-v2 /usr/local/bin/
RUN ctr oci spec \
        | jq '.hooks.createContainer[.hooks.createContainer| length] |= . + {"path": "/kind/bin/mount-product-files.sh"}' \
        | jq 'del(.process.rlimits)' \
        > /etc/containerd/cri-base.json \
    && containerd --version
COPY --from=build-containerd /_LICENSES/* /LICENSES/
# copy over runc build and install
COPY --from=build-runc /runc/runc /usr/local/sbin/runc
RUN runc --version
COPY --from=build-runc /_LICENSES/* /LICENSES/
# copy over crictl build and install
COPY --from=build-crictl /cri-tools/build/crictl /usr/local/bin/
COPY --from=build-crictl /_LICENSES/* /LICENSES/
# copy over CNI plugins build and install
RUN  mkdir -p /opt/cni/bin
COPY --from=build-cni /cni-plugins/bin/host-local /opt/cni/bin/
COPY --from=build-cni /cni-plugins/bin/loopback /opt/cni/bin/
COPY --from=build-cni /cni-plugins/bin/ptp /opt/cni/bin/
COPY --from=build-cni /cni-plugins/bin/portmap /opt/cni/bin/
COPY --from=build-cni /_LICENSES/* /LICENSES/
# copy over containerd-fuse-overlayfs and install
COPY --from=build-fuse-overlayfs /fuse-overlayfs-snapshotter/bin/containerd-fuse-overlayfs-grpc /usr/local/bin/
COPY --from=build-fuse-overlayfs /_LICENSES/* /LICENSES/

# squash down to one compressed layer, without any lingering whiteout files etc
FROM scratch
COPY --from=build / /
# add metadata, must be done after the squashing
# first tell systemd that it is in docker (it will check for the container env)
# https://systemd.io/CONTAINER_INTERFACE/
ENV container docker
# systemd exits on SIGRTMIN+3, not SIGTERM (which re-executes it)
# https://bugzilla.redhat.com/show_bug.cgi?id=1201657
STOPSIGNAL SIGRTMIN+3
# NOTE: this is *only* for documentation, the entrypoint is overridden later
ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]
