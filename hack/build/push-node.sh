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

set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# place to stick temp binaries
BINDIR="${REPO_ROOT}/_output/bin"

# install kind from the repo into $BINDIR
get_kind() {
  # build kind from the repo and use that ...
  GOBIN="${BINDIR}" go install .
  echo "${BINDIR}/kind"
}

# select kind binary to use
KIND="${KIND:-$(get_kind)}"

# generate tag
DATE="$(date +v%Y%m%d)"
TAG="${DATE}-$(git describe --tags --always --dirty)"

# build
set -x
"${KIND}" build node-image --image="kindest/node:${TAG}" --base-image="kindest/base:${TAG}"

# re-tag with kubernetes version
IMG="kindest/node:${TAG}"
KUBE_VERSION="$(docker run --rm --entrypoint=cat "${IMG}" /kind/version)"
docker tag "${IMG}" "kindest/node:${KUBE_VERSION}"

# push
docker push kindest/node:"${KUBE_VERSION}"


