#!/usr/bin/env bash
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

# simple script to build binaries for release

set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# controls the number of concurrent builds
PRALLELISM=${PRALLELISM:-6}

echo "Building in parallel for:"
# What we do here:
# - use xargs to build in parallel (-P) while collecting a combined exit code
# - use cat to supply the individual args to xargs (one line each)
# - use env -S to split the line into environment variables and execute
# - ... the build
# NOTE: the binary name needs to in single quotes so we delay evaluating
# GOOS / GOARCH
if xargs -n1 -P "${PRALLELISM}" -I{} \
    env -S {} make build 'KIND_BINARY_NAME=kind-${GOOS}-${GOARCH}'; then
    echo "Cross build passed!" 1>&2
else
    echo "Cross build failed!" 1>&2
    exit 1
fi < <(cat <<EOF
GOOS=windows GOARCH=amd64
GOOS=darwin GOARCH=amd64
GOOS=linux GOARCH=amd64
GOOS=linux GOARCH=arm
GOOS=linux GOARCH=arm64
GOOS=linux GOARCH=ppc64le
EOF
)
