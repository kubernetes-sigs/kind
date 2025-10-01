#!/bin/bash
set -eux -o pipefail
# Ensure network-related modules to be loaded
modprobe tap ip_tables iptable_nat ip6_tables ip6table_nat

# The moby-engine package included in Fedora lacks support for rootless,
# So we need to install docker-ce and docker-ce-rootless-extras from the upstream.
DNF_REPO=""
INSTALL_PODMAN="1"
if grep -q centos /etc/os-release; then
	# Works with Rocky and Alma too
	DNF_REPO="https://download.docker.com/linux/centos/docker-ce.repo"
	if grep -q el8 /etc/os-release; then
		# podman seems to conflict with docker-ce on EL8
		INSTALL_PODMAN=""
	fi
elif grep -q fedora /etc/os-release; then
	DNF_REPO="https://download.docker.com/linux/fedora/docker-ce.repo"
else
	echo >&2 "Unsupported OS"
	exit 1
fi
DNF="dnf"
if command -v dnf5 &>/dev/null; then
	# DNF 5 (Fedora 41 or later)
	DNF="dnf5"
	"$DNF" config-manager addrepo --from-repofile="${DNF_REPO}"
else
	# DNF 4
	"$DNF" config-manager --add-repo="${DNF_REPO}"
fi
"$DNF" install -y git golang make docker-ce docker-ce-rootless-extras
systemctl enable --now docker
if [ -n "${INSTALL_PODMAN}" ]; then
	"$DNF" install -y podman
fi

# Install kubectl
GOARCH="$(uname -m | sed -e 's/aarch64/arm64/' -e 's/x86_64/amd64/')"
curl -L -o /usr/bin/kubectl "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${GOARCH}/kubectl"
chmod +x /usr/bin/kubectl

# Configuration for rootless: https://kind.sigs.k8s.io/docs/user/rootless/
mkdir -p "/etc/systemd/system/user@.service.d"
cat <<EOF >"/etc/systemd/system/user@.service.d/delegate.conf"
[Service]
Delegate=yes
EOF
systemctl daemon-reload
