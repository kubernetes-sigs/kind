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

# run golangci-lint
GO111MODULE=on bin/golangci-lint \
  --enable=golint --enable=vet --enable=gofmt \
  --enable=misspell --enable=gochecknoinits \
  run ./pkg/... ./cmd/... .
