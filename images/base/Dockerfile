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

# kind cluster base image, built on Kubernetes's debian-base
# which is basically a very stripped down debian image (~43MB)
# that ships with a small utility 'clean-install' which is an ideal drop in for
# apt-get update && apt-get install $@ 
# see https://github.com/kubernetes/kubernetes/tree/master/build/debian-base
#
# To this we add systemd, CNI, and other tools needed to run Kubeadm
#
# For systemd + docker configuration used below, see the following references:
# https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
# https://developers.redhat.com/blog/2014/05/05/running-systemd-within-docker-container/
# https://developers.redhat.com/blog/2016/09/13/running-systemd-in-a-non-privileged-container/

# TODO(bentheelder): explore supporting other arches, IE support the ARCH arg.
# For now this is still an ARG so we can reuse it throughout the build
# without persisting it to the container.
ARG ARCH="amd64"
ARG BASE_IMAGE_VERSION="0.3.2"
ARG BASE_IMAGE="k8s.gcr.io/debian-base-${ARCH}:${BASE_IMAGE_VERSION}"
FROM ${BASE_IMAGE}

# NOTE: ARCH must be defined again after FROM
# https://docs.docker.com/engine/reference/builder/#scope
ARG ARCH="amd64"

# setting DEBIAN_FRONTEND=noninteractive stops some apt warnings, this is not 
# a real argument, we're (ab)using ARG to get a temporary ENV again.
ARG DEBIAN_FRONTEND=noninteractive
# Get dependencies
# The base image already has: ssh, apt, snapd
# This is broken down into:
# - packages necessary for installing docker
# - packages needed to run services (systemd)
# - packages needed for docker / hyperkube / kubernetes components
# - misc packages (utilities we use in our own tooling)
# Then we cleanup (removing unwanted systemd services)
# https://developers.redhat.com/blog/2014/05/05/running-systemd-within-docker-container/
RUN clean-install \
      apt-transport-https ca-certificates curl software-properties-common gnupg2 lsb-release \
      systemd systemd-sysv \
      iptables iproute2 ethtool socat util-linux mount ebtables udev kmod aufs-tools \
      bash rsync \
    && find /lib/systemd/system/sysinit.target.wants/ -name "systemd-tmpfiles-setup.service" -delete \
    && rm -f /lib/systemd/system/multi-user.target.wants/* \
    && rm -f /etc/systemd/system/*.wants/* \
    && rm -f /lib/systemd/system/local-fs.target.wants/* \
    && rm -f /lib/systemd/system/sockets.target.wants/*udev* \
    && rm -f /lib/systemd/system/sockets.target.wants/*initctl* \
    && rm -f /lib/systemd/system/basic.target.wants/*

# Install docker, which needs to happen after we install some of the packages above
# based on https://docs.docker.com/install/linux/docker-ce/ubuntu/#set-up-the-repository
# and https://kubernetes.io/docs/setup/independent/install-kubeadm/#installing-docker
# - get docker's GPG key
# - add the fingerprint
# - add the repository
# - update apt, install docker, cleanup
# NOTE: 17.03 is officially supported by Kubernetes currently, so we pin to that.
# https://kubernetes.io/docs/tasks/tools/install-kubeadm/
ARG DOCKER_VERSION="17.03.2~ce-0~debian-stretch"
# another temporary env, not a real argument. setting this to a non-zero value
# silences this warning from apt-key:
# "Warning: apt-key output should not be parsed (stdout is not a terminal)"
ARG APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE="false"
RUN curl -fsSL "https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg" | apt-key add - \
    && apt-key fingerprint 0EBFCD88 \
    && ARCH="${ARCH}" add-apt-repository \
        "deb [arch=${ARCH}] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable" \
    && clean-install "docker-ce=${DOCKER_VERSION}"

# Install CNI binaries to /opt/cni/bin
# TODO(bentheelder): doc why / what here
ARG CNI_VERSION="0.6.0"
ARG CNI_TARBALL="cni-plugins-${ARCH}-v${CNI_VERSION}.tgz"
ARG CNI_BASE_URL="https://storage.googleapis.com/kubernetes-release/network-plugins/"
ARG CNI_URL="${CNI_BASE_URL}${CNI_TARBALL}"
RUN curl -sSL --retry 5 --output /tmp/cni.tgz "${CNI_URL}" \
    && sha256sum /tmp/cni.tgz \
    && mkdir -p /opt/cni/bin \
    && tar -C /opt/cni/bin -xzf /tmp/cni.tgz

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
# We need systemd to be PID1 to run the various services (docker, kubelete, etc.)
ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]

# TODO(bentheelder): add a systemd unit to run our node bootup logic
# https://bugzilla.redhat.com/show_bug.cgi?id=809132
# TODO(bentheelder): deal with systemd MAC address assignment
# https://github.com/systemd/systemd/issues/3374#issuecomment-288882355
# https://github.com/systemd/systemd/issues/3374#issuecomment-339258483
