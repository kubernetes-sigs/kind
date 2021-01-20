#!/usr/bin/env bash
# Copyright 2020 The Kubernetes Authors.
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
set -x;

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# pass through git details from prow / image builder
if [ -n "${PULL_BASE_SHA:-}" ]; then
  export COMMIT="${PULL_BASE_SHA:?}"
else
  COMMIT="$(git rev-parse HEAD 2>/dev/null)"
  export COMMIT
fi
# short commit is currently 8 characters
SHORT_COMMIT="${COMMIT:0:8}"

# we upload here
BUCKET="${BUCKET:-k8s-staging-kind}"
# under each of these
VERSIONS=(
  latest
  "${SHORT_COMMIT}"
)

# build for all platforms
hack/release/build/cross.sh

# upload to the bucket
for f in bin/kind-*; do
  # make a tarball with this
  # TODO: eliminate e2e-k8s.sh
  base="$(basename "$f")"
  platform="${base#kind-}"
  tar \
    -czvf "bin/${platform}.tgz" \
    --transform 's#.*kind.*#kind#' \
    --transform 's#.*e2e-k8s.sh#e2e-k8s.sh#' \
    --transform='s#^/#./#' \
    --mode='755' \
    "${f}" \
    "hack/ci/e2e-k8s.sh"
  
  # copy everything up to each version
  for version in "${VERSIONS[@]}"; do
    gsutil cp -P "bin/${platform}.tgz" "gs://${BUCKET}/${version}/${platform}.tgz"
    gsutil cp -P "$f" "gs://${BUCKET}/${version}/${base}"
  done
done

# upload the e2e script so kubernetes CI can consume it
for version in "${VERSIONS[@]}"; do
  gsutil cp -P hack/ci/e2e-k8s.sh "gs://${BUCKET}/${version}/e2e-k8s.sh"
done
