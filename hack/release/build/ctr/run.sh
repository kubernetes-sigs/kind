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

# simple script to run cloudbuilds for ctr for each arch kind supports
# NOTE: this is temporary, we only need to build ctr until our (tiny!)
# --no-unpack patch is available in the standard package

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# options
PROJECT="${PROJECT:-bentheelder-kind-dev}"
BUCKET="${BUCKET:-bentheelder-kind-dev/containerd}"
# default to the merge commit for:
# https://github.com/containerd/containerd/pull/3259
# ->
# https://github.com/containerd/containerd/commit/d71c7ada27959fe04fad5390367e4fab215334b3
CONTAINERD_SOURCE="${CONTAINERD_SOURCE:-$(go env GOPATH)/src/github.com/containerd/containerd}"
CONTAINERD_REF="${CONTAINERD_REF:-d71c7ada27959fe04fad5390367e4fab215334b3}"
CTR_CLOUDBUILD="${REPO_ROOT}/hack/release/build/ctr/cloudbuild.yaml"

# make sure we submit the build to the right project
gcloud config set core/project "${PROJECT}"

# get current working directory to be the desired containerd ref
cd "${CONTAINERD_SOURCE}"
git fetch --all
git checkout "${CONTAINERD_REF}"

# TODO(bentheelder): this is hacky, compute what the makefile would have computed
# so we can just go build the one package with our own flags
# https://github.com/containerd/containerd/blob/a17c8095716415cebb1157a27db5fccace56b0fc/Makefile#L22-L24
VERSION=$(git describe --match 'v[0-9]*' --dirty='.m' --always)
REVISION=$(git rev-parse HEAD)$(if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)

# submit a build for each arch
GOARCHES=(
  "arm"
  "arm64"
  "amd64"
  "ppc64le"
)
for arch in "${GOARCHES[@]}"; do
  gcloud builds submit \
    --config="${CTR_CLOUDBUILD}" \
    --substitutions="_GOARCH=${arch},_BUCKET=${BUCKET},_VERSION=${VERSION},_REVISION=${REVISION}" \
    .
done
