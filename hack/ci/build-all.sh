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

# simple script to ensure all of our builds work

set -o errexit
set -o nounset
set -o pipefail
set -o verbose

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

exit_trap() {
  # we _want_ to check the exit code directly
  # shellcheck disable=SC2181
  if [ "$?" -eq 0 ]; then
    echo "Build passed!"
  else
    echo "Build Failed!"
  fi
}

trap exit_trap EXIT

# build go binaries without bazzel
go install .
