#!/bin/sh
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

# Simple posix sh reproducible build container script with (go) caching
# Only requires docker on the host
set -o nounset
set -o errexit

# get and go to the repo root
REPO_ROOT="${REPO_ROOT:-$(git rev-parse --show-toplevel)}"

# autodetect host GOOS and GOARCH if not set, even if go is not installed
GOOS="${GOOS:-$("${REPO_ROOT}/hack/build/goos.sh")}"
GOARCH="${GOARCH:-$("${REPO_ROOT}/hack/build/goarch.sh")}"

# use the official module proxy by default
GOPROXY="${GOPROXY:-https://proxy.golang.org}"

# default build image
GO_VERSION="1.12.6"
GO_IMAGE="golang:${GO_VERSION}"

# docker volume name, used as a go module / build cache
CACHE_VOLUME="kind-build-cache"

# output directory
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/bin}"
# source directory
SOURCE_DIR="${SOURCE_DIR:-${REPO_ROOT}}"

# creates the output directory
make_out_dir() {
  mkdir -p "${OUT_DIR}"
}

# creates the cache volume
make_cache_volume() {
  docker volume create "${CACHE_VOLUME}" >/dev/null
}

# runs $@ in a go container with caching etc. and the repo mount to /src
run_in_go_container() {
  # get user id and group id so we can run the container
  _UID=$(id -u)
  _GID=$(id -g)
  # run in the container
  docker run \
    `# ensure the container is removed on exit` \
      --rm \
    `# use the cache volume for go` \
      -v "${CACHE_VOLUME}:/go" \
      -e GOCACHE=/go/cache \
    `# mount the output & repo dir, set working directory to the repo` \
      -v "${OUT_DIR}:/out" \
      -v "${SOURCE_DIR}:/src" \
      -w "/src" \
    `# go settings: use modules and proxy, disable cgo, set OS/Arch` \
      -e GO111MODULE=on \
      -e GOPROXY="${GOPROXY}" \
      -e CGO_ENABLED=0 \
      -e GOOS="${GOOS}" \
      -e GOARCH="${GOARCH}" \
    `# pass through proxy settings from the host` \
      -e HTTP_PROXY \
      -e HTTPS_PROXY \
      -e NO_PROXY \
    `# run as if the host user for consistent file permissions` \
      --user "${_UID}:${_GID}" \
    `# use the golang image for the container` \
      "${GO_IMAGE}" \
    `# and finally, pass through args` \
      "$@"
}

make_out_dir
make_cache_volume
run_in_go_container "$@"
