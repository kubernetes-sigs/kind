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

set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

# build for all platforms
hack/release/build/cross.sh

# upload to latest bucket
for f in bin/kind-*; do
  # NOTE: this bucket is temporary until prow is migrated to the CNCF
  # this is just a google-hosted bucket used specifically for kind
  # periodic UNTRUSTED CI builds used to speed up kubernetes CI
  gsutil cp "$f" "gs://bentheelder-kind-ci-builds/latest/$(basename "$f")"
done

# upload the e2e script so kubernetes CI can consume it
gsutil cp hack/ci/e2e-k8s.sh gs://bentheelder-kind-ci-builds/latest/e2e-k8s.sh
