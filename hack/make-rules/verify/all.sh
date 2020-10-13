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

set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# exit code, if a script fails we'll set this to 1
res=0

# run all verify scripts, optionally skipping any of them

if [[ "${VERIFY_LINT:-true}" == "true" ]]; then
  echo "verifying lints ..."
  hack/make-rules/verify/lint.sh || res=1
  cd "${REPO_ROOT}"
fi

if [[ "${VERIFY_GENERATED:-true}" == "true" ]]; then
  echo "verifying generated ..."
  hack/make-rules/verify/generated.sh || res=1
  cd "${REPO_ROOT}"
fi

if [[ "${VERIFY_SHELLCHECK:-true}" == "true" ]]; then
  echo "verifying shellcheck ..."
  hack/make-rules/verify/shellcheck.sh || res=1
  cd "${REPO_ROOT}"
fi

# exit based on verify scripts
if [[ "${res}" = 0 ]]; then
  echo ""
  echo "All verify checks passed, congrats!"
else
  echo ""
  echo "One or more verify checks failed! See output above..."
fi
exit "${res}"

