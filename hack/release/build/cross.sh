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

# cd to the repo root and setup go
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"
source hack/build/setup-go.sh

# controls the number of concurrent builds
PARALLELISM=${PARALLELISM:-6}

echo "Building in parallel for:"
# What we do here:
# - use xargs to build in parallel (-P) while collecting a combined exit code
# - use cat to supply the individual args to xargs (one line each)
# - use env -S to split the line into environment variables and execute
# - ... the build
# NOTE: the binary name needs to be in single quotes so we delay evaluating
# GOOS / GOARCH
# NOTE: disable SC2016 because we _intend_ for these to evaluate later
# shellcheck disable=SC2016
if xargs -0 -n1 -P "${PARALLELISM}" bash -c 'eval $0; make build KIND_BINARY_NAME=kind-${GOOS}-${GOARCH}'; then
    echo "Cross build passed!" 1>&2
else
    echo "Cross build failed!" 1>&2
    exit 1
fi < <(cat <<EOF | tr '\n' '\0'
export GOOS=windows GOARCH=amd64
export GOOS=darwin GOARCH=amd64
export GOOS=darwin GOARCH=arm64
export GOOS=linux GOARCH=amd64
export GOOS=linux GOARCH=arm64
EOF
)

# add sha256 for binaries
cd "${REPO_ROOT}"/bin
for f in kind-*; do
    shasum -a 256 "$f" > "$f".sha256sum;
done
