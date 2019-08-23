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

set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
cd "${REPO_ROOT}"

# ensure we have up to date kind
make build

# generate tag
TAG="$(date +v%Y%m%d)-$(git describe --always --dirty)"
IMAGE="kindest/base:${TAG}"

# build
(set -x; "${REPO_ROOT}/bin/kind" build base-image --image="${IMAGE}" --source="${REPO_ROOT}/images/base/")

# push
docker push "${IMAGE}"
