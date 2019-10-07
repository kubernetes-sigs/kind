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

# kind cluster base image
#
# For systemd + docker configuration used below, see the following references:
# https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
# https://developers.redhat.com/blog/2014/05/05/running-systemd-within-docker-container/
# https://developers.redhat.com/blog/2016/09/13/running-systemd-in-a-non-privileged-container/

ARG BASE_IMAGE="ubuntu:19.10"
FROM ${BASE_IMAGE}

# setting DEBIAN_FRONTEND=noninteractive stops some apt warnings, this is not 
# a real argument, we're (ab)using ARG to get a temporary ENV again.
ARG DEBIAN_FRONTEND=noninteractive

COPY clean-install /usr/local/bin/clean-install
RUN chmod +x /usr/local/bin/clean-install

# Get dependencies
# The base image already has: ssh, apt, snapd
# This is broken down into (each on a line):
# - packages needed to run services (systemd)
# - CRI (containerd)
# - packages needed for kubernetes components
# - misc packages kind uses itself
# Then we cleanup (removing unwanted systemd services)
# Then we disable kmsg in journald (these log entries would be confusing)
#
# ********************************<TEMPORARY>***********************************
#
# We then download a ctr binary with a tiny additional feature
# we use that is not yet merged / packaged.
#
# See: https://github.com/containerd/containerd/pull/3259
#
# This binary is built with hack/build/ctr/run.sh
# Sources: https://github.com/BenTheElder/containerd/tree/kind
# Additional commit:
# https://github.com/BenTheElder/containerd/commit/cb7c780af2394ab08d5d8a3932ca7437074ae179
#
# TODO(bentheelder): remove this once --no-unpack is packaged upstream.
#
# *******************************</TEMPORARY>***********************************
#
# https://developers.redhat.com/blog/2014/05/05/running-systemd-within-docker-container/
RUN clean-install \
      systemd systemd-sysv libsystemd0 \
      conntrack iptables iproute2 ethtool socat util-linux mount ebtables udev kmod \
      bash ca-certificates curl rsync \
    && find /lib/systemd/system/sysinit.target.wants/ -name "systemd-tmpfiles-setup.service" -delete \
    && rm -f /lib/systemd/system/multi-user.target.wants/* \
    && rm -f /etc/systemd/system/*.wants/* \
    && rm -f /lib/systemd/system/local-fs.target.wants/* \
    && rm -f /lib/systemd/system/sockets.target.wants/*udev* \
    && rm -f /lib/systemd/system/sockets.target.wants/*initctl* \
    && rm -f /lib/systemd/system/basic.target.wants/* \
    && echo "ReadKMsg=no" >> /etc/systemd/journald.conf \
    && echo "done installing packages"

# override the rp_filter settings to enable calico cni to "just work"
COPY 10-network-security.conf /etc/sysctl.d/

# Install containerd and runc binaries from kind-ci/containerd-nightlies repository
# The repository contains latest stable releases and nightlies using golang's pseudo-version
# CONTAINERD_VERSION=0.0.0-yyyymmddhhmmss-commitid
ARG CONTAINERD_VERSION="0.0.0-20191007143651-efd38f48"
ARG CONTAINERD_BASE_URL="https://github.com/kind-ci/containerd-nightlies/releases/download/"
RUN export ARCH=$(dpkg --print-architecture | sed 's/ppc64el/ppc64le/' | sed 's/armhf/arm/') \
    && export CONTAINERD_TARBALL="v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}.linux-${ARCH}.tar.gz" \
    && export CONTAINERD_URL="${CONTAINERD_BASE_URL}${CONTAINERD_TARBALL}" \
    && curl -sSL --retry 5 --output /tmp/containerd.tgz "${CONTAINERD_URL}" \
    && tar -C /usr/local -xzf /tmp/containerd.tgz \
    && rm -rf /tmp/containerd.tgz \
    && export RUNC_URL="${CONTAINERD_BASE_URL}v${CONTAINERD_VERSION}/runc.${ARCH}" \
    && curl -sSL --retry 5 --output /usr/local/sbin/runc "${RUNC_URL}" \
    && chmod 755 /usr/local/sbin/runc

# Install containerd systemd unit file
COPY containerd.service /etc/systemd/system
RUN systemctl enable containerd
# debug containerd version and create default config
# additionally:
# - disable some plugins we don't use / support
# - setup the "test-handler" used by kubernetes runtime class testing
RUN containerd --version \
    && mkdir -p /etc/containerd \
    && echo "# disable plugins we don't / can't support" > /etc/containerd/config.toml \
    && echo 'disabled_plugins = ["aufs", "btrfs", "zfs"]' >> /etc/containerd/config.toml \
    && echo '' >> /etc/containerd/config.toml \
    && echo '# Runtime handler used for runtime class test.' >> /etc/containerd/config.toml \
    && echo '[plugins.cri.containerd.runtimes.test-handler]' >> /etc/containerd/config.toml \
    && echo '  runtime_type = "io.containerd.runtime.v1.linux"' >> /etc/containerd/config.toml

# Install CNI binaries to /opt/cni/bin
# TODO(bentheelder): doc why / what here
ARG CNI_VERSION="v0.8.2"
ARG CNI_BASE_URL="https://github.com/containernetworking/plugins/releases/download/"
RUN export ARCH=$(dpkg --print-architecture | sed 's/ppc64el/ppc64le/' | sed 's/armhf/arm/') \
    && export CNI_TARBALL="${CNI_VERSION}/cni-plugins-linux-${ARCH}-${CNI_VERSION}.tgz" \
    && export CNI_URL="${CNI_BASE_URL}${CNI_TARBALL}" \
    && curl -sSL --retry 5 --output /tmp/cni.tgz "${CNI_URL}" \
    && mkdir -p /opt/cni/bin \
    && tar -C /opt/cni/bin -xzf /tmp/cni.tgz \
    && rm -rf /tmp/cni.tgz

# Install crictl to /usr/local/bin
ARG CRICTL_VERSION="v1.16.1"
RUN export ARCH=$(dpkg --print-architecture | sed 's/ppc64el/ppc64le/' | sed 's/armhf/arm/') \
    && curl -fSL "https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_VERSION}/crictl-${CRICTL_VERSION}-linux-${ARCH}.tar.gz" | tar xzC /usr/local/bin \
    && echo 'runtime-endpoint: unix:///var/run/containerd/containerd.sock' > /etc/crictl.yaml

# tell systemd that it is in docker (it will check for the container env)
# https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
ENV container docker
# systemd exits on SIGRTMIN+3, not SIGTERM (which re-executes it)
# https://bugzilla.redhat.com/show_bug.cgi?id=1201657
STOPSIGNAL SIGRTMIN+3

# wrap systemd with our special entrypoint, see pkg/build for how this is built
# basically this just lets us set up some things before continuing on to systemd
# while preserving that systemd is PID1
# for how we leverage this, see pkg/cluster
COPY [ "entrypoint", "/usr/local/bin/" ]
# We need systemd to be PID1 to run the various services (docker, kubelet, etc.)
# NOTE: this is *only* for documentation, the entrypoint is overridden by the node image
ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]
