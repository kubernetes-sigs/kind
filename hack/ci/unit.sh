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

# script to run unit tests, with coverage enabled and junit xml output
set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"

# build gotestsum
SOURCE_DIR="${REPO_ROOT}/hack/tools" GOOS=linux hack/go_container.sh \
  go build -o /out/gotestsum gotest.tools/gotestsum

# run unit tests with coverage enabled and junit output
GOOS=linux hack/go_container.sh \
  /out/gotestsum --junitfile=/out/junit.xml -- \
    -coverprofile=/out/unit.cov -covermode count -coverpkg sigs.k8s.io/... ./...

# filter out generated files
sed '/zz_generated/d' bin/unit.cov > bin/filtered.cov

# generate cover html
hack/go_container.sh go tool cover -html=/out/filtered.cov -o /out/filtered.html

# if we are in CI, copy to the artifact upload location
if [[ -n "${ARTIFACTS:-}" ]]; then
  cp bin/junit.xml bin/filtered.cov bin/filtered.html "${ARTIFACTS:?}/"
fi
