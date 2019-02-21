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

set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# create a temporary directory
TMP_DIR=$(mktemp -d)

# cleanup
exitHandler() (
  echo "Cleaning up ..."
  rm -rf "${TMP_DIR}"
)
trap exitHandler EXIT

# pull misspell
export GO111MODULE=on
URL="https://github.com/client9/misspell"
echo "Cloning ${URL} in ${TMP_DIR}..."
git clone --quiet --depth=1 "${URL}" "${TMP_DIR}"
pushd "${TMP_DIR}" > /dev/null
go mod init
popd > /dev/null

# build misspell
BIN_PATH="${TMP_DIR}/cmd/misspell"
pushd "${BIN_PATH}" > /dev/null
echo "Building misspell ..."
go build > /dev/null
popd > /dev/null

# check spelling
RES=0
ERROR_LOG="${TMP_DIR}/errors.log"
echo "Checking spelling ..."
git ls-files | grep -v -e vendor | xargs "${BIN_PATH}/misspell" > "${ERROR_LOG}"
if [[ -s "${ERROR_LOG}" ]]; then
  sed 's/^/error: /' "${ERROR_LOG}" # add 'error' to each line to highlight in e2e status
  echo "Found spelling errors!"
  RES=1
else
  echo "Done!"
fi
exit "${RES}"
