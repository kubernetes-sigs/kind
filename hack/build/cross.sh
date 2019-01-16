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

set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

OUT="${REPO_ROOT}/_output/bin"
mkdir -p "${OUT}"

CLEAN="false"
for i in "$@" ; do
    if [[ $i == "--clean" ]] ; then
        CLEAN="true"
        break
    fi
done

if [[ "${CLEAN}" == "true" ]]; then
    echo "Cleaning ${OUT}/kind-*"
    rm -f "${OUT}/kind-*"
fi

build() {
    GOOS="${1}"
    GOARCH="${2}"
    export GOOS
    export GOARCH
    # build without CGO for cross compiling and distributing
    CGO_ENABLED=0
    export CGO_ENABLED
    local out_path
    out_path="${OUT}/kind-${GOOS}-${GOARCH}"
    echo "${out_path}"
    go build -o "${out_path}" sigs.k8s.io/kind
}

# TODO(bentheelder): support more platforms
echo "Building in parallel for:"
build "linux" "amd64" & \
build "darwin" "amd64" & \
build "windows" "amd64" & \
wait
