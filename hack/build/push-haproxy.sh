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

# builds and pushes the haproxy image for all architectures

set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# cd to the image
cd ./images/haproxy

IMAGE="${IMAGE:-kindest/haproxy}"
TAG="${TAG:-0.1.0}"

ARCHES=(
  "amd64,amd64"
  "arm,arm32v6"
  "arm64,arm64v8"
  "ppc64le,ppc64le"
)

# build all images
images=()
for arch in "${ARCHES[@]}"; do
  # split arch pair
  build_arch="$(sed -r 's/,.*//' <<< "${arch}")"
  tag_arch="$(sed -r 's/.*,//' <<< "${arch}")"
  # build image
  image="${IMAGE}:${build_arch}-${TAG}"
  docker build "--build-arg=ARCH=${tag_arch}" -f Dockerfile -t "${image}" .
  docker push "${image}"
  images+=("${image}")
done

# This option is required for running the docker manifest command
export DOCKER_CLI_EXPERIMENTAL="enabled"

# create and push the manifest
docker manifest create "${IMAGE}:${TAG}" "${images[@]}"
for image in "${images[@]}"; do
    # image:arch-tag, grab arch
    arch="$(sed -r 's/.*://; s/-.*//' <<< "${image}")"
    docker manifest annotate "${IMAGE}:${TAG}" "${image}" --arch "${arch}"
done
docker manifest push -p "${IMAGE}:{TAG}"
