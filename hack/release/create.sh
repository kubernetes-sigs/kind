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

# creates a release and following pre-release commit for `kind`
# builds binaries between the commits
# Use like: create.sh <release-version> <next-prerelease-version>
# EG: create.sh 0.3.0 0.4.0
set -o errexit -o nounset -o pipefail

UPSTREAM='https://github.com/kubernetes-sigs/kind.git'

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${REPO_ROOT}"

# check for arguments
if [ "$#" -ne 2 ]; then
    echo "Usage: create.sh release-version next-prerelease-version"
    exit 1
fi

# darwin is great
SED="sed"
if which gsed &>/dev/null; then
  SED="gsed"
fi
if ! (${SED} --version 2>&1 | grep -q GNU); then
  echo "!!! GNU sed is required.  If on OS X, use 'brew install gnu-sed'." >&2
  exit 1
fi

VERSION_FILE="./pkg/cmd/kind/version/version.go"

# update core version in go code to $1 and pre-release version to $2
set_version() {
  ${SED} -i "s/versionCore = .*/versionCore = \"${1}\"/" "${VERSION_FILE}"
  ${SED} -i "s/versionPreRelease = .*/versionPreRelease = \"${2}\"/" "${VERSION_FILE}"
  echo "Updated ${VERSION_FILE} for ${1}"
}

# make a commit denoting the version ($1)
make_commit() {
  git add "${VERSION_FILE}"
  git commit -m "version ${1}"
  echo "Created commit for ${1}"
}

# add a git tag with $1
add_tag() {
  git tag "${1}"
  echo "Tagged ${1}"
}

# create the first version, tag and build it
set_version "${1}" ""
make_commit "v${1}"
add_tag "v${1}"
echo "Building ..."
make clean && ./hack/release/build/cross.sh

# update to the second version
set_version "${2}" "alpha"
make_commit "v${2}-alpha"
add_tag "v${2}-alpha"

# print follow-up instructions
echo ""
echo "Created commits for v${1} and v${2}, you should now:"
echo " - git push"
echo " - File a PR with these pushed commits"
echo " - Merge the PR"
echo " - git push ${UPSTREAM} v${1}"
echo " - Create a GitHub release from the pushed tag v${1}"
