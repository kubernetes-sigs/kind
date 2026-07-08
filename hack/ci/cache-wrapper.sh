#!/bin/bash
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

# USAGE: cache-wrapper.sh hack/go_container.sh some-go-command
# $@ will be executed with this script wrapping cache upload / download

set -o errexit -o nounset -o pipefail

# options for where the cache is stored
BUCKET="${BUCKET:-bentheelder-kind-ci-builds}"
BRANCH="${BRANCH:-main}"
CACHE_SUFFIX="${CACHE_SUFFIX:-"ci-cache/${BRANCH}/gocache.tar"}"
CACHE_URL="https://storage.googleapis.com/${BUCKET}/${CACHE_SUFFIX}"
CACHE_GS="gs://${BUCKET}/${CACHE_SUFFIX}"

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"

# default to downloading cache
if [ "${DOWNLOAD_CACHE:-true}" = "true" ]; then
  # NOTE:
  # - We clean the modcache because we won't be able to write to it if it exists
  # https://github.com/golang/go/issues/31481
  # - All of the relevant go system directories are under /go in KIND's build
  # - See below for how the cache tarball is created
  hack/go_container.sh sh -c "go clean -modcache && curl -sSL ${CACHE_URL} | tar -C /go -zxf - --overwrite"
fi

# run the supplied command and store the exit code for later
set +o errexit
"$@"
res=$?
set -o errexit

# default to not uploading cache
if [ "${UPLOAD_CACHE:-false}" = "true" ]; then
  # We want to cache:
  # - XDG_CACHE_HOME / the go build cache, this is /go/cache in KIND's build
  # - The module cache, ~= $GOPATH/pkg/mod. this /go/pkg/mod in KIND's build
  hack/go_container.sh sh -c 'tar -C /go -czf - ./cache ./pkg/mod' | gsutil cp - "${CACHE_GS}"
fi

# preserve the exit code from our real task
exit $res
