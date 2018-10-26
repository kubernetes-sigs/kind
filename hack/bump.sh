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

# simple script to bump the cli version

set -o errexit
set -o nounset
set -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"


main () {
    TYPE="${TYPE:-patch}"
    VERSION="$(go run hack/bumpversion/main.go --type="${TYPE}" --prerelease="alpha" 2>&1)"
    export VERSION
    # regenerate Version constant
    hack/update-generated.sh

    # commit the new version
    git commit -am "bump version to ${VERSION}"
    git tag "${VERSION}"
}

main
