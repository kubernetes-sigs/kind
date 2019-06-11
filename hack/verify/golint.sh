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

# enable modules and the proxy cache
export GO111MODULE="on"
GOPROXY="${GOPROXY:-https://proxy.golang.org}"
export GOPROXY

# build golint
BINDIR="${REPO_ROOT}/bin"
# use the tools module
cd "hack/tools"
go build -o "${BINDIR}/golint" golang.org/x/lint/golint
# go back to the root
cd "${REPO_ROOT}"

"${BINDIR}/golint" -set_exit_status ./pkg/... ./cmd/... .
