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

set -o errexit
set -o nounset
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

main() {
  # run vendor update script in dry run mode
  diff=$("${DEP}" check || true)
  if [[ -n "${diff}" ]]; then
    echo "Non-zero output from dep check" >&2
    echo "" >&2
    echo "${diff}" >&2
    echo "" >&2
    echo "please run hack/update-deps.sh" >&2
    exit 1
  fi
}

main
