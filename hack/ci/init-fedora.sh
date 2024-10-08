#!/bin/bash
set -eux -o pipefail
# Ensure network-related modules to be loaded
modprobe tap ip_tables iptable_nat ip6_tables ip6table_nat

# The moby-engine package included in Fedora lacks support for rootless,
# So we need to install docker-ce and docker-ce-rootless-extras from the upstream.
curl -fsSL https://get.docker.com | sh
dnf install -y golang-go make kubernetes-client podman docker-ce-rootless-extras
systemctl enable --now docker

# Configuration for rootless: https://kind.sigs.k8s.io/docs/user/rootless/
mkdir -p "/etc/systemd/system/user@.service.d"
cat <<EOF >"/etc/systemd/system/user@.service.d/delegate.conf"
[Service]
Delegate=yes
EOF
systemctl daemon-reload
