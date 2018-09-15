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

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# run vendor update script
hack/update-deps.sh

# make sure the tree is clean
status="$(git status -s)"
if [[ -n "${status}" ]]; then
  echo "unexpectedly dirty working directory after hack/update-deps.sh"
  echo "${status}"
  echo ""
  echo "please run and commit: hack/update-deps.sh"
  exit 1
fi

