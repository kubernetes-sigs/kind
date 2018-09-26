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

# 'go generate's kind, using tools from vendor (go-bindata)
set -o nounset
set -o errexit
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

# install go-bindata from vendor locally
OUTPUT_GOBIN="${REPO_ROOT}/_output/bin"
cd "${REPO_ROOT}"
GOBIN="${OUTPUT_GOBIN}" go install ./vendor/github.com/jteeuwen/go-bindata/go-bindata
GOBIN="${OUTPUT_GOBIN}" go install ./vendor/k8s.io/code-generator/cmd/defaulter-gen
GOBIN="${OUTPUT_GOBIN}" go install ./vendor/k8s.io/code-generator/cmd/deepcopy-gen

# go generate (using go-bindata)
# NOTE: go will only take package paths, not absolute directories
export PATH="${OUTPUT_GOBIN}:${PATH}"
go generate ./...
deepcopy-gen -i ./pkg/cluster/config/ -O zz_generated.deepcopy --go-header-file boilerplate.go.txt
defaulter-gen -i ./pkg/cluster/config -O zz_generated.default --go-header-file boilerplate.go.txt

# gofmt the tree
find . -path "./vendor" -prune -o -name "*.go" -type f -print0 | xargs -0 gofmt -s -w
