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

REGISTRY="${REGISTRY:-kindest}"

# cd to the repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}"

# ensure we have up to date kind
make build

# path to kubernetes sources
KUBEROOT="${KUBEROOT:-"$(go env GOPATH)"/src/k8s.io/kubernetes}"

# ensure we have qemu setup (de-duped logic with setting up buildx for multi-arch)
"${REPO_ROOT}/hack/build/init-buildx.sh"

# NOTE: adding platforms is costly in terms of build time
# we will consider expanding this in the future, for now the aim is to prove
# multi-arch and enable developers working on commonly available hardware
# Other users are free to build their own images on additional platforms using
# their own time and resources. Please see our docs.
ARCHES="${ARCHES:-amd64 arm64}"
IFS=" " read -r -a __arches__ <<< "$ARCHES"

set -x
# ensure clean build
(cd "${KUBEROOT}" && make clean)
# get kubernetes version
version_line="$(cd "${KUBEROOT}"; ./hack/print-workspace-status.sh | grep 'gitVersion')"
kube_version="${version_line#"STABLE_DOCKER_TAG "}"

# kubernetes build option(s)
GOFLAGS="${GOFLAGS:-}"
if [ -z "${GOFLAGS}" ]; then
    # TODO: dockerless only applies to < 1.24, the version selection here is brittle
    case "${kube_version}" in
    v1.1[0-8].*)
        GOFLAGS="-tags=providerless"
        ;;
    *)
        GOFLAGS="-tags=providerless,dockerless"
        ;;
    esac
fi
export GOFLAGS

# build for each arch
IMAGE="${REGISTRY}/node:${kube_version}"
images=()
for arch in "${__arches__[@]}"; do
    image="${REGISTRY}/node-${arch}:${kube_version}"
    "${REPO_ROOT}/bin/kind" build node-image --image="${image}" --arch="${arch}" "${KUBEROOT}"
    images+=("${image}")
done

# combine to manifest list tagged with kubernetes version
export DOCKER_CLI_EXPERIMENTAL=enabled
# images must be pushed to be referenced by docker manifest
# we push only after all builds have succeeded
for image in "${images[@]}"; do
    docker push "${image}"
done
docker manifest rm "${IMAGE}" || true
docker manifest create "${IMAGE}" "${images[@]}"
docker manifest push "${IMAGE}"
