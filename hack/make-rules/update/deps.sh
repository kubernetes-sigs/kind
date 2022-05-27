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

# Runs go mod tidy, go mod vendor, and then prunes vendor
#
# Usage:
#   deps.sh
set -o errexit -o nounset -o pipefail

# cd to the repo root and setup go
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"
source hack/build/setup-go.sh

# tidy all modules
go mod tidy

cd "${REPO_ROOT}/hack/tools"
go mod tidy

# NOTE: kindnetd is only built for linux and uses linux APIs
cd "${REPO_ROOT}/images/kindnetd"
GOOS=linux go mod tidy
