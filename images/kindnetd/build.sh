#!/usr/bin/env bash
# Copyright 2019 The Kubernetes Authors.
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

set -o nounset
set -o errexit
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# build the binary
export GOARCH="${GOARCH:-amd64}"
export GOOS="linux"
export SOURCE_DIR="${REPO_ROOT}/cmd/kindnetd"
# NOTE: use a per-arch OUT_DIR so we send less in the docker build context
export OUT_DIR="${REPO_ROOT}/bin/kindnetd/${GOARCH}"
hack/build/go_container.sh go build -v -o /out/kindnetd

# TODO: verisoning
# build image
IMAGE="${IMAGE:-kindest/kindnetd}"
TAG="${TAG:-$(cat images/kindnetd/VERSION)}"
docker build \
  -t "${IMAGE}:${TAG}" \
  --build-arg="GOARCH=${GOARCH}" \
  -f images/kindnetd/Dockerfile \
  "${OUT_DIR}"
