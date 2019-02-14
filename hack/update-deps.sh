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


# Runs go mod tidy, go mod vendor, and then prun vendor
#
# Usage:
#   update-deps.sh

set -o nounset
set -o errexit
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

prune-vendor() {
  find vendor -type f \
    -not -iname "*.c" \
    -not -iname "*.go" \
    -not -iname "*.h" \
    -not -iname "*.proto" \
    -not -iname "*.s" \
    -not -iname "AUTHORS*" \
    -not -iname "CONTRIBUTORS*" \
    -not -iname "COPYING*" \
    -not -iname "LICENSE*" \
    -not -iname "NOTICE*" \
    -exec rm '{}' \;
}

export GO111MODULE="on"
go mod tidy
go mod vendor
prune-vendor

# TODO(bentheelder): re-enable once (if?) we're using docker client
# docker has a contrib dir with nothing we use in it, dep will retain only the
# licenses, so we manually prune this directory.
# See https://github.com/kubernetes/steering/issues/57
#rm -rf vendor/github.com/docker/docker/contrib
