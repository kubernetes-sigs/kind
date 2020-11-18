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

# script to run unit / integration tests, with coverage enabled and junit xml output
set -o errexit -o nounset -o pipefail

# cd to the repo root and setup go
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"
source hack/build/setup-go.sh

# set to 'unit' or 'integration' to run a subset
MODE="${MODE:-all}"

# build gotestsum
cd 'hack/tools'
go build -o "${REPO_ROOT}/bin/gotestsum" gotest.tools/gotestsum
cd "${REPO_ROOT}"

go_test_opts=(
  "-coverprofile=${REPO_ROOT}/bin/${MODE}.cov"
  '-covermode' 'count'
  '-coverpkg' 'sigs.k8s.io/kind/...'
)
if [[ "${MODE}" = 'unit' ]]; then
  go_test_opts+=('-short' '-tags=nointegration')
elif [[ "${MODE}" = 'integration' ]]; then
  go_test_opts+=('-run' '^TestIntegration')
fi

# run unit tests with coverage enabled and junit output
(
  set -x; 
  "${REPO_ROOT}/bin/gotestsum" --junitfile="${REPO_ROOT}/bin/${MODE}-junit.xml" \
    -- "${go_test_opts[@]}" './...'
)

# filter out generated files
sed '/zz_generated/d' "${REPO_ROOT}/bin/${MODE}.cov" > "${REPO_ROOT}/bin/${MODE}-filtered.cov"

# generate cover html
go tool cover -html="${REPO_ROOT}/bin/${MODE}-filtered.cov" -o "${REPO_ROOT}/bin/${MODE}-filtered.html"

# if we are in CI, copy to the artifact upload location
if [[ -n "${ARTIFACTS:-}" ]]; then
  cp "bin/${MODE}-junit.xml" "${ARTIFACTS:?}/junit.xml"
  cp "${REPO_ROOT}/bin/${MODE}-filtered.cov" "${ARTIFACTS:?}/filtered.cov"
  cp "${REPO_ROOT}/bin/${MODE}-filtered.html" "${ARTIFACTS:?}/filtered.html"
fi
