#!/bin/bash
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
set -o xtrace

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# place to stick temp binaries
BINDIR="${REPO_ROOT}/_output/bin"

# obtain dep either from existing bazel build (in case of running in an sh_binary)
# or install it from vendor into BINDIR
get_dep() {
    # look for local bazel built dep first
    local dep
    dep="$(find bazel-bin/ -type f -name dep | head -n1)"
    # if we found dep from bazel, just return that
    if [[ -n "${dep}" ]]; then
        echo "dep"
        return 0
    fi
    # otherwise build dep from vendor and use that ...
    GOBIN="${BINDIR}" go install ./vendor/github.com/golang/dep/cmd/dep
    echo "${BINDIR}/dep"
}

# select dep binary to use
DEP="${DEP:-$(get_dep)}"


# dep itself has a problematic testdata directory with infinite symlinks which
# makes bazel sad: https://github.com/golang/dep/pull/1412
# dep should probably be removing it, but it doesn't:
# https://github.com/golang/dep/issues/1580
rm -rf vendor/github.com/golang/dep/internal/fs/testdata
# go-bindata does too, and is not maintained ...
rm -rf vendor/github.com/jteeuwen/go-bindata/testdata
# docker has a contrib dir with nothing we use in it, dep will retain only the
# licenses, which includes some GPL, so we manually prune this directory.
# See https://github.com/kubernetes/steering/issues/57
rm -rf vendor/github.com/docker/docker/contrib

# actually run dep
"${DEP}" ensure -v "$@"

# rm all of the junk again
rm -rf vendor/github.com/golang/dep/internal/fs/testdata
rm -rf vendor/github.com/jteeuwen/go-bindata/testdata
rm -rf vendor/github.com/docker/docker/contrib

# update BUILD since we may have updated vendor/...
hack/update-bazel.sh

echo SUCCESS
