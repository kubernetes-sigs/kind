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

REPO_ROOT="${REPO_ROOT:-$(git rev-parse --show-toplevel)}"
cd "${REPO_ROOT}"

# enable modules and the proxy cache
export GO111MODULE="on"
GOPROXY="${GOPROXY:-https://proxy.golang.org}"
export GOPROXY

# build the generators
BINDIR="${REPO_ROOT}/bin"
# use the tools module
cd "hack/tools"
go build -o "${BINDIR}/defaulter-gen" k8s.io/code-generator/cmd/defaulter-gen
go build -o "${BINDIR}/deepcopy-gen" k8s.io/code-generator/cmd/deepcopy-gen
go build -o "${BINDIR}/conversion-gen" k8s.io/code-generator/cmd/conversion-gen
# go back to the root
cd "${REPO_ROOT}"

# turn off module mode before running the generators
# https://github.com/kubernetes/code-generator/issues/69
# we also need to populate vendor
go mod tidy
go mod vendor
export GO111MODULE="off"

# fake being in a gopath
FAKE_GOPATH="$(mktemp -d)"
trap 'rm -rf ${FAKE_GOPATH}' EXIT

FAKE_REPOPATH="${FAKE_GOPATH}/src/sigs.k8s.io/kind"
mkdir -p "$(dirname "${FAKE_REPOPATH}")" && ln -s "${REPO_ROOT}" "${FAKE_REPOPATH}"

export GOPATH="${FAKE_GOPATH}"
cd "${FAKE_REPOPATH}"

# run the generators
"${BINDIR}/deepcopy-gen" -i ./pkg/internal/apis/config/ -O zz_generated.deepcopy --go-header-file hack/boilerplate.go.txt
"${BINDIR}/defaulter-gen" -i ./pkg/internal/apis/config/ -O zz_generated.default --go-header-file hack/boilerplate.go.txt

"${BINDIR}/deepcopy-gen" -i ./pkg/apis/config/v1alpha3 -O zz_generated.deepcopy --go-header-file hack/boilerplate.go.txt
"${BINDIR}/defaulter-gen" -i ./pkg/apis/config/v1alpha3 -O zz_generated.default --go-header-file hack/boilerplate.go.txt
"${BINDIR}/conversion-gen" -i ./pkg/internal/apis/config/v1alpha3 -O zz_generated.conversion --go-header-file hack/boilerplate.go.txt

export GO111MODULE="on"
cd "${REPO_ROOT}"

# gofmt the tree
find . -name "*.go" -type f -print0 | xargs -0 gofmt -s -w
