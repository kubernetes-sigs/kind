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
set -x

# Function: Upload file to GCS bucket
upload_file() {
    local file="$1"
    local version="$2"
    gsutil -m cp -P "${file}" "gs://${BUCKET}/${version}/$(basename "${file}")"
}

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# pass through git details from prow / image builder
if [ -n "${PULL_BASE_SHA:-}" ]; then
    export COMMIT="${PULL_BASE_SHA:?}"
else
    COMMIT="$(git rev-parse HEAD 2>/dev/null)"
    export COMMIT
fi
# short commit is currently 8 characters
SHORT_COMMIT="${COMMIT:0:8}"

# we upload here
BUCKET="${BUCKET:-k8s-staging-kind}"
# under each of these
VERSIONS=(
    "latest"
    "${SHORT_COMMIT}"
)

# Create temporary directory and ensure cleanup
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "${TEMP_DIR}"' EXIT

echo "Building for all platforms"
hack/release/build/cross.sh

echo "Uploading to bucket: ${BUCKET}"
for f in bin/kind-*; do
    base="$(basename "${f}")"
    platform="${base#kind-}"
    
    echo "Processing ${platform}"
    tar \
        -czvf "${TEMP_DIR}/${platform}.tgz" \
        --transform 's#.*kind.*#kind#' \
        --transform 's#.*e2e-k8s.sh#e2e-k8s.sh#' \
        --transform='s#^/#./#' \
        --mode='755' \
        "${f}" \
        "hack/ci/e2e-k8s.sh"
    
    # copy everything up to each version
    for version in "${VERSIONS[@]}"; do
        upload_file "${TEMP_DIR}/${platform}.tgz" "${version}"
        upload_file "${f}" "${version}"
    done
done

echo "Uploading e2e script"
for version in "${VERSIONS[@]}"; do
    upload_file "hack/ci/e2e-k8s.sh" "${version}"
done

echo "Upload process completed successfully"
