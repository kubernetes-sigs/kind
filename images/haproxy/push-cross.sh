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
TAG="${TAG:-2.0.0-alpine}"
BASE="haproxy:${TAG}"

# tag arch, manifest architecture, variant
ARCHES=(
  "amd64,amd64,"
  "arm32v6,arm,v6"
  "arm64v8,arm64,v8"
  "ppc64le,ppc64le,"
)

# build all images
images=()
image_infos=()
for arch_info in "${ARCHES[@]}"; do
  # build image
  tag_arch="$(sed -E 's#^([^,]+),[^,]*,[^,]*$#\1#' <<<"${arch_info}")"
  image="${IMAGE}:${tag_arch}-${TAG}"
  docker build "--build-arg=ARCH=${tag_arch}" "--build-arg=BASE=${BASE}" -f Dockerfile -t "${image}" .
  docker push "${image}"
  # join image we tagged with arch info for building the manifest later 
  images+=("${image}")
  image_infos+=("${image},${arch_info}")
done

# This option is required for running the docker manifest command
export DOCKER_CLI_EXPERIMENTAL="enabled"

# create and push the manifest
docker manifest create "${IMAGE}:${TAG}" "${images[@]}"
for image_info in "${image_infos[@]}"; do
  # split out image info
  # image:arch-tag, grab arch
  image="$(sed -E 's#^([^,]+),[^,]+,[^,]+,[^,]*$#\1#' <<<"${image_info}")"
  #tag_arch="$(sed -E 's#^[^,]+,([^,]+),[^,]+,[^,]*$#\1#' <<<"${image_info}")"
  architecture="$(sed -E 's#^[^,]+,[^,]+,([^,]+),[^,]*$#\1#' <<<"${image_info}")"
  variant="$(sed -E 's#^[^,]+,[^,]+,[^,]+,([^,]*)$#\1#' <<<"${image_info}")"
  args=("--arch" "${architecture}")
  if [ -n "${variant}" ]; then
    args+=("--variant" "${variant}")
  fi
  # add the image to the manifest
  docker manifest annotate "${IMAGE}:${TAG}" "${image}" "${args[@]}"
done
docker manifest push --purge "${IMAGE}:${TAG}"
