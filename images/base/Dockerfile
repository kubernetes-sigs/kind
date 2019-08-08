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

ARG BASE_IMAGE="ubuntu:19.04"
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
      containerd \
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
    && systemctl enable containerd \
    && export ARCH=$(dpkg --print-architecture | sed 's/ppc64el/ppc64le/' | sed 's/armhf/arm/') \
    && curl -fSL -o /usr/local/bin/ctr \
      "https://storage.googleapis.com/bentheelder-kind-dev/containerd/linux/${ARCH}/ctr" \
    && chmod +x /usr/local/bin/ctr \
    && echo "done installing packages"

# add overrides to containerd
COPY 10-limits.conf 10-restart.conf /etc/systemd/system/containerd.service.d/

# debug containerd version and create default config
# additionally, disable some plugins we don't use / support
RUN containerd --version \
    && mkdir -p /etc/containerd \
    && echo 'disabled_plugins = ["aufs", "btrfs", "zfs"]' > /etc/containerd/config.toml \
    && containerd config default >> /etc/containerd/config.toml

# Install CNI binaries to /opt/cni/bin
# TODO(bentheelder): doc why / what here
ARG CNI_VERSION="0.7.5"
ARG CNI_BASE_URL="https://storage.googleapis.com/kubernetes-release/network-plugins/"
RUN export ARCH=$(dpkg --print-architecture | sed 's/ppc64el/ppc64le/' | sed 's/armhf/arm/') \
    && export CNI_TARBALL="cni-plugins-${ARCH}-v${CNI_VERSION}.tgz" \
    && export CNI_URL="${CNI_BASE_URL}${CNI_TARBALL}" \
    && curl -sSL --retry 5 --output /tmp/cni.tgz "${CNI_URL}" \
    && sha256sum /tmp/cni.tgz \
    && mkdir -p /opt/cni/bin \
    && tar -C /opt/cni/bin -xzf /tmp/cni.tgz \
    && rm -rf /tmp/cni.tgz

# Install crictl to /usr/local/bin
ARG CRICTL_VERSION="v1.14.0"
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
