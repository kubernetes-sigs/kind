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

# get the repo root for defaulting OUT_DIR and SOURCE_DIR
# we assume the repo root is two levels up from this script
REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"

# ============================ SCRIPT SETTINGS =================================
# output directory, will be mounted to /out, defaults to /bin in REPO_ROOT
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/bin}"
# source directory, will be mounted to /src, defaults to REPO_ROOT
SOURCE_DIR="${SOURCE_DIR:-${REPO_ROOT}}"
# GOPROXY is respected by go, use the official module proxy by default
# this helps make our build more reproducible and reliable
GOPROXY="${GOPROXY:-https://proxy.golang.org}"
# the container image, by default a recent official golang image
GOIMAGE="${GOIMAGE:-golang:1.12.9}"
# ========================== END SCRIPT SETTINGS ===============================

# autodetects and host GOOS and GOARCH and sets them if not set
# works even if go is not installed on the host
detect_and_set_goos_goarch() {
  # if we have go, just ask go! NOTE: this respects explicitly set GOARCH / GOOS
  if which go >/dev/null 2>&1; then
    GOARCH=$(go env GOARCH)
    GOOS=$(go env GOOS)
    return
  fi

  # detect GOOS equivalent if unset
  if [ -z "${GOOS:-}" ]; then
    case "$(uname -s)" in
      Darwin) GOOS="darwin" ;;
      Linux) GOOS="linux" ;;
      *) echo "Unknown host OS! '$(uname -s)'" exit 2 ;;
    esac
  fi

  # detect GOARCH equivalent if unset
  if [ -z "${GOARCH:-}" ]; then
    case "$(uname -m)" in
      x86_64) GOARCH="amd64" ;;
      arm*)
        GOARCH="arm"
        if [ "$(getconf LONG_BIT)" = "64" ]; then
          GOARCH="arm64"
        fi
      ;;
      *) echo "Unknown host architecture! '$(uname -m)'" exit 2 ;;
    esac
  fi
}

# creates the output directory
make_out_dir() {
  mkdir -p "${OUT_DIR}"
}

# docker volume name, used as a go module / build cache
__CACHE_VOLUME__="kind-build-cache"

# creates the cache volume
make_cache_volume() {
  docker volume create "${__CACHE_VOLUME__}" >/dev/null
}

# runs $@ in a go container with caching etc. and the repo mount to /src
run_in_go_container() {
  # get host user id and group id so we can run the container as them
  # this ensures sane file ownership for build outputs
  _UID=$(id -u)
  _GID=$(id -g)
  # run in the container
  docker run \
    `# ensure the container is removed on exit` \
      --rm \
    `# use the cache volume for go` \
      -v "${__CACHE_VOLUME__}:/go" \
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
      "${GOIMAGE}" \
    `# and finally, pass through args` \
      "$@"
}

make_out_dir
make_cache_volume
detect_and_set_goos_goarch
run_in_go_container "$@"
