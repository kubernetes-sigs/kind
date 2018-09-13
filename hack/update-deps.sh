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

REPO_ROOT=$(git rev-parse --show-toplevel)

trap 'echo "FAILED" >&2' ERR

# TODO(bentheelder): support first-class using a bazel-built dep instead
DEP="dep"

cd "${REPO_ROOT}"


# dep itself has a problematic testdata directory with infinite symlinks which
# makes bazel sad: https://github.com/golang/dep/pull/1412
# dep should probably be removing it, but it doesn't:
# https://github.com/golang/dep/issues/1580
rm -rf vendor/github.com/golang/dep/internal/fs/testdata
# go-bindata does too, and is not maintained ...
rm -rf vendor/github.com/jteeuwen/go-bindata/testdata
# docker has a contrib dir with nothing we use in it, dep will retain the licenses
# which includes some GPL, so we manually prune this. 
# See https://github.com/kubernetes/steering/issues/57
rm -rf vendor/github.com/docker/docker/contrib
"${DEP}" ensure -v "$@"
rm -rf vendor/github.com/golang/dep/internal/fs/testdata
rm -rf vendor/github.com/jteeuwen/go-bindata/testdata
rm -rf vendor/github.com/docker/docker/contrib
hack/update-bazel.sh
echo SUCCESS
