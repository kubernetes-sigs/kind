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

# CI script to run go lint over our code
set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# place to stick temp binaries
BINDIR="${REPO_ROOT}/_output/bin"
mkdir -p "${BINDIR}"

# TMP_GOPATH is used in get_golint
TMP_GOPATH="$(TMPDIR="${BINDIR}" mktemp -d "${BINDIR}/verify-deps.XXXXX")"
trap 'rm -rf "${TMP_GOPATH}"' EXIT

# install golint from vendor into $BINDIR
get_golint() {
  GOBIN="${BINDIR}" go install ./vendor/github.com/golang/lint/golint
  echo "${BINDIR}/golint"
}

# select golint binary to use
GOLINT="${GOLINT:-$(get_golint)}"

# we need to do this because golint ./... matches vendor...
go list ./... | xargs -L1 "${GOLINT}" -set_exit_status
