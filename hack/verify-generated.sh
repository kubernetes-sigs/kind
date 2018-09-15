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

# script to verify generated files
set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

bazel build //pkg/build/sources:bindata

BAZEL_GENERATED_BINDATA="bazel-genfiles/pkg/build/sources/images_sources.go"
GO_GENERATED_BINDATA="pkg/build/sources/images_sources.go"

DIFF="$(diff <(cat "${GO_GENERATED_BINDATA}") <(gofmt -s "${BAZEL_GENERATED_BINDATA}"))"
if [ ! -z "$DIFF" ]; then
  echo "${GO_GENERATED_BINDATA} does not match ${BAZEL_GENERATED_BINDATA}"
  echo "please run and commit: hack/generate.sh"
  echo "if you have changed the generation, please ensure these remain identical" 
  echo "see: hack/bindata.bzl, pkg/build/sources/BUILD.bazel, pkg/build/sources/generate.go"
  exit 1
fi
