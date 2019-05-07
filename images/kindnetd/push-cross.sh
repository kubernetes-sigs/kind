#!/bin/bash
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

# builds and pushes the kindnetd image for all architectures

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

IMAGE="${IMAGE:-kindest/kindnetd}"
TAG="${TAG:-$(cat images/kindnetd/VERSION)}"

ARCHES=(
  "amd64"
  "arm"
  "arm64"
  "ppc64le"
)

# darwin is great
SED="sed"
if which gsed &>/dev/null; then
  SED="gsed"
fi
if ! (${SED} --version 2>&1 | grep -q GNU); then
  echo "!!! GNU sed is required.  If on OS X, use 'brew install gnu-sed'." >&2
  exit 1
fi

# build all images
images=()
for arch in "${ARCHES[@]}"; do
  # build image
  tag="${arch}-${TAG}"
  GOARCH="${arch}" TAG="${tag}" images/kindnetd/build.sh
  docker push "${IMAGE}:${tag}"
  images+=("${IMAGE}:${tag}")
done

# This option is required for running the docker manifest command
export DOCKER_CLI_EXPERIMENTAL="enabled"

# create and push the manifest
docker manifest create "${IMAGE}:${TAG}" "${images[@]}"
for image in "${images[@]}"; do
    # image:arch-tag, grab arch
    arch="$(${SED} -r 's/.*://; s/-.*//' <<< "${image}")"
    docker manifest annotate "${IMAGE}:${TAG}" "${image}" --os "linux" --arch "${arch}"
done
docker manifest push -p "${IMAGE}:${TAG}"
