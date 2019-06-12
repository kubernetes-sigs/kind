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

# ensure we have up to date kind
make build
KIND="${REPO_ROOT}/bin/kind"

# generate tag
DATE="$(date +v%Y%m%d)"
TAG="${DATE}-$(git describe --always --dirty)"

# build
KUBEROOT="${KUBEROOT:-${GOPATH}/src/k8s.io/kubernetes}"
set -x
"${KIND}" build node-image --image="kindest/node:${TAG}" --kube-root="${KUBEROOT}"

# re-tag with kubernetes version
IMG="kindest/node:${TAG}"
KUBE_VERSION="$(docker run --rm --entrypoint=cat "${IMG}" /kind/version)"
docker tag "${IMG}" "kindest/node:${KUBE_VERSION}"

# push
docker push kindest/node:"${KUBE_VERSION}"
