#!/usr/bin/env bash
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

# script to run linters
set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"

# build golangci-lint
SOURCE_DIR="${REPO_ROOT}/hack/tools" hack/go_container.sh \
  go build -o /out/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
SOURCE_DIR="${REPO_ROOT}/hack/tools" GOOS="linux" hack/go_container.sh \
  go build -o /out/golangci-lint-linux github.com/golangci/golangci-lint/cmd/golangci-lint

# run golangci-lint
LINTS=(
  # default golangci-lint lints
  deadcode errcheck gosimple govet ineffassign staticcheck \
  structcheck typecheck unused varcheck \
  # additional lints
  golint gofmt misspell gochecknoinits unparam scopelint
)
ENABLE=$(sed 's/ /,/g' <<< "${LINTS[@]}")
# first for the repo in general
GO111MODULE=on bin/golangci-lint --disable-all --enable="${ENABLE}" run ./pkg/... ./cmd/... .
# ... and then for kindnetd, which is only on linux
SOURCE_DIR="${REPO_ROOT}/images/kindnetd" GOOS="linux" "${REPO_ROOT}/hack/go_container.sh" \
  /out/golangci-lint-linux --disable-all --enable="${ENABLE}" run ./...
