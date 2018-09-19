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


# Run dep ensure and generate bazel rules.
#
# Usage:
#   update-deps.sh <ARGS>
#
# The args are sent to dep ensure -v <ARGS>

set -o nounset
set -o errexit
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# place to stick temp binaries
BINDIR="${REPO_ROOT}/_output/bin"

# install dep from vendor into $BINDIR
get_dep() {
  # build dep from vendor-fake-gopath and use that ...
  GOBIN="${BINDIR}" go install ./vendor/github.com/golang/dep/cmd/dep
  echo "${BINDIR}/dep"
}

# select dep binary to use
DEP="${DEP:-$(get_dep)}"

# TODO(bentheelder): re-enable once (if?) we're using docker client
# docker has a contrib dir with nothing we use in it, dep will retain only the
# licenses, so we manually prune this directory.
# See https://github.com/kubernetes/steering/issues/57
#rm -rf vendor/github.com/docker/docker/contrib

# actually run dep
"${DEP}" ensure -v "$@"

# TODO(bentheelder): re-enable once (if?) we're using docker client
# we have to rm it again after dep ensure
#rm -rf vendor/github.com/docker/docker/contrib
