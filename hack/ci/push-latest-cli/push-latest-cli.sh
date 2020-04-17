#!/usr/bin/env bash
# Copyright 2020 The Kubernetes Authors.
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
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd -P)"
cd "${REPO_ROOT}"

# pass through git details from prow / image builder
if [ -n "${PULL_BASE_SHA:-}" ]; then
  export COMMIT="${PULL_BASE_SHA:?}"
fi

# build for all platforms
hack/release/build/cross.sh

# upload to latest bucket
BUCKET="${BUCKET:-k8s-staging-kind}"
for f in bin/kind-*; do
  # NOTE: this bucket is temporary until prow is migrated to the CNCF
  # this is just a google-hosted bucket used specifically for kind
  # periodic UNTRUSTED CI builds used to speed up kubernetes CI
  gsutil cp -P "$f" "gs://${BUCKET}/latest/$(basename "$f")"
done

# upload the e2e script so kubernetes CI can consume it
gsutil cp -P hack/ci/e2e-k8s.sh "gs://${BUCKET}/latest/e2e-k8s.sh"
