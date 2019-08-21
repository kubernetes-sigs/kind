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
# EG: create.sh v0.3.0 v0.4.0-alpha
set -o nounset
set -o errexit
set -o pipefail

if [ "$#" -ne 2 ]; then
    echo "Usage: create.sh release-version next-prerelease-version"
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# darwin is great
SED="sed"
if which gsed &>/dev/null; then
  SED="gsed"
fi
if ! (${SED} --version 2>&1 | grep -q GNU); then
  echo "!!! GNU sed is required.  If on OS X, use 'brew install gnu-sed'." >&2
  exit 1
fi

VERSION_FILE="./cmd/kind/version/version.go"

# update version in go code to $1
set_version() {
    ${SED} -i "s/Version = .*/Version = \"${1}\"/" "${VERSION_FILE}"
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
set_version "${1}"
make_commit "${1}"
add_tag "${1}"
echo "Building ..."
make clean
./hack/release/build/cross.sh --clean

# update to the second version
set_version "${2}"
make_commit "${2}"

# print follow-up instructions
echo ""
echo "Created commits for ${1} and ${2}, you should now:"
echo " - File a PR with these commits"
echo " - Merge the PR"
echo " - git push upstream ${1}"
echo " - create a GitHub release from ${1}"
