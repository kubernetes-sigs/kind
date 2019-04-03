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

# kind cluster base image, built on ubuntu:18.04
#
# To this we add systemd, CNI, and other tools needed to run Kubeadm
#
# For systemd + docker configuration used below, see the following references:
# https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
# https://developers.redhat.com/blog/2014/05/05/running-systemd-within-docker-container/
# https://developers.redhat.com/blog/2016/09/13/running-systemd-in-a-non-privileged-container/

ARG BASE_IMAGE="ubuntu:18.04"
FROM ${BASE_IMAGE}

# setting DEBIAN_FRONTEND=noninteractive stops some apt warnings, this is not 
# a real argument, we're (ab)using ARG to get a temporary ENV again.
ARG DEBIAN_FRONTEND=noninteractive

COPY clean-install /usr/local/bin/clean-install
RUN chmod +x /usr/local/bin/clean-install

# Get dependencies
# The base image already has: ssh, apt, snapd
# This is broken down into (each on a line):
# - packages necessary for installing docker
# - packages needed to run services (systemd)
# - packages needed for docker / hyperkube / kubernetes components
# - misc packages (utilities we use in our own tooling)
# Then we cleanup (removing unwanted systemd services)
# Finally we disable kmsg in journald
# https://developers.redhat.com/blog/2014/05/05/running-systemd-within-docker-container/
RUN clean-install \
      apt-transport-https ca-certificates curl software-properties-common gnupg2 lsb-release \
      systemd systemd-sysv libsystemd0 \
      conntrack iptables iproute2 ethtool socat util-linux mount ebtables udev kmod aufs-tools \
      bash rsync \
    && find /lib/systemd/system/sysinit.target.wants/ -name "systemd-tmpfiles-setup.service" -delete \
    && rm -f /lib/systemd/system/multi-user.target.wants/* \
    && rm -f /etc/systemd/system/*.wants/* \
    && rm -f /lib/systemd/system/local-fs.target.wants/* \
    && rm -f /lib/systemd/system/sockets.target.wants/*udev* \
    && rm -f /lib/systemd/system/sockets.target.wants/*initctl* \
    && rm -f /lib/systemd/system/basic.target.wants/* \
    && echo "ReadKMsg=no" >> /etc/systemd/journald.conf

# Install docker, which needs to happen after we install some of the packages above
# based on https://docs.docker.com/install/linux/docker-ce/ubuntu/#set-up-the-repository
# and https://kubernetes.io/docs/setup/independent/install-kubeadm/#installing-docker
# - get docker's GPG key
# - add the fingerprint
# - add the repository
# - update apt, install docker, cleanup
# NOTE: 18.09 is officially supported by Kubernetes currently, so we pin to that.
# https://kubernetes.io/docs/tasks/tools/install-kubeadm/
ARG DOCKER_VERSION="5:18.09.*"
# another temporary env, not a real argument. setting this to a non-zero value
# silences this warning from apt-key:
# "Warning: apt-key output should not be parsed (stdout is not a terminal)"
ARG APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE="false"
RUN curl -fsSL "https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg" | apt-key add - \
    && apt-key fingerprint 0EBFCD88 \
    && add-apt-repository \
        "deb https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable" \
    && clean-install "docker-ce=${DOCKER_VERSION}"

# Enable CRI for containerd
RUN mkdir -p /etc/docker && echo '{"cri-containerd": true}' > /etc/docker/daemon.json
RUN mkdir -p /etc/containerd && containerd config default > /etc/containerd/config.toml

# Install CNI binaries to /opt/cni/bin
# TODO(bentheelder): doc why / what here
ARG CNI_VERSION="0.7.5"
ARG CNI_BASE_URL="https://storage.googleapis.com/kubernetes-release/network-plugins/"
RUN export ARCH=$(dpkg --print-architecture) \
    && export CNI_TARBALL="cni-plugins-${ARCH}-v${CNI_VERSION}.tgz" \
    && export CNI_URL="${CNI_BASE_URL}${CNI_TARBALL}" \
    && curl -sSL --retry 5 --output /tmp/cni.tgz "${CNI_URL}" \
    && sha256sum /tmp/cni.tgz \
    && mkdir -p /opt/cni/bin \
    && tar -C /opt/cni/bin -xzf /tmp/cni.tgz \
    && rm -rf /tmp/cni.tgz

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
COPY [ "entrypoint/entrypoint", "/usr/local/bin/" ]
# We need systemd to be PID1 to run the various services (docker, kubelet, etc.)
# NOTE: this is *only* for documentation, the entrypoint is overridden at runtime
ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]

# the docker graph must be a volume to avoid overlay on overlay
# NOTE: we do this last because changing a volume with a Dockerfile must
# occur before defining it.
# See: https://docs.docker.com/engine/reference/builder/#volume
VOLUME [ "/var/lib/docker" ]

# TODO(bentheelder): deal with systemd MAC address assignment
# https://github.com/systemd/systemd/issues/3374#issuecomment-288882355
# https://github.com/systemd/systemd/issues/3374#issuecomment-339258483
