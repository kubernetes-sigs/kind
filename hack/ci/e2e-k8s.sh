#!/bin/sh
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

# hack script for running a kind e2e
# must be run with a kubernetes checkout in $PWD (IE from the checkout)
# Usage: SKIP="ginkgo skip regex" FOCUS="ginkgo focus regex" kind-e2e.sh

set -o errexit -o nounset -o xtrace

# Settings:
# SKIP: ginkgo skip regex
# FOCUS: ginkgo focus regex
# BUILD_TYPE: bazel or make
# 

# our exit handler (trap)
cleanup() {
  # KIND_CREATE_ATTEMPTED is true once we: kind create
  if [ "${KIND_CREATE_ATTEMPTED:-}" = true ]; then
    kind "export" logs "${ARTIFACTS}/logs" || true
    kind delete cluster || true
  fi
  rm -f _output/bin/e2e.test || true
  # remove our tempdir, this needs to be last, or it will prevent kind delete
  [ -n "${TMP_DIR:-}" ] && rm -rf "${TMP_DIR:?}"
}

# build kubernetes / node image, e2e binaries, with bazel
build_with_bazel() {
  # possibly enable bazel build caching before building kubernetes
  if [ "${BAZEL_REMOTE_CACHE_ENABLED:-false}" = "true" ]; then
    create_bazel_cache_rcs.sh || true
  fi

  # build the node image w/ kubernetes
  kind build node-image --type=bazel
  # make sure we have e2e requirements
  bazel build //cmd/kubectl //test/e2e:e2e.test //vendor/github.com/onsi/ginkgo/ginkgo

  # ensure the e2e script will find our binaries ...
  # https://github.com/kubernetes/kubernetes/issues/68306
  # TODO: remove this, it was fixed in 1.13+
  mkdir -p '_output/bin/'
  cp 'bazel-bin/test/e2e/e2e.test' '_output/bin/'
  PATH="$(dirname "$(find "${PWD}/bazel-bin/" -name kubectl -type f)"):${PATH}"
  export PATH
}

# build kubernetes / node image, e2e binaries
build() {
  # build the node image w/ kubernetes
  kind build node-image
  # make sure we have e2e requirements
  make all WHAT='cmd/kubectl test/e2e/e2e.test vendor/github.com/onsi/ginkgo/ginkgo'
}

# up a cluster with kind
create_cluster() {
  # create the config file
  cat <<EOF > "${ARTIFACTS}/kind-config.yaml"
# config for 1 control plane node and 2 workers (necessary for conformance)
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
networking:
  ipFamily: ${IP_FAMILY:-ipv4}
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
  # NOTE: must match the number of workers above
  NUM_NODES=2
  # actually create the cluster
  # TODO(BenTheElder): settle on verbosity for this script
  KIND_CREATE_ATTEMPTED=true
  kind create cluster \
    --image=kindest/node:latest \
    --retain \
    --wait=1m \
    -v=3 \
    "--config=${ARTIFACTS}/kind-config.yaml"
}

# run e2es with ginkgo-e2e.sh
run_tests() {
  # export the KUBECONFIG
  KUBECONFIG="$(kind get kubeconfig-path)"
  export KUBECONFIG

  # IPv6 clusters need some CoreDNS changes in order to work in k8s CI:
  # 1. k8s CI doesn´t offer IPv6 connectivity, so CoreDNS should be configured
  # to work in an offline environment:
  # https://github.com/coredns/coredns/issues/2494#issuecomment-457215452
  # 2. k8s CI adds following domains to resolv.conf search field:
  # c.k8s-prow-builds.internal google.internal.
  # CoreDNS should handle those domains and answer with NXDOMAIN instead of SERVFAIL
  # otherwise pods stops trying to resolve the domain.
  if [ "${IP_FAMILY:-ipv4}" = "ipv6" ]; then
    # Get the current config
    original_coredns=$(kubectl get -oyaml -n=kube-system configmap/coredns)
    echo "Original CoreDNS config:"
    echo "${original_coredns}"
    # Patch it
    fixed_coredns=$(
      printf '%s' "${original_coredns}" | sed \
        -e 's/^.*kubernetes cluster\.local/& internal/' \
        -e '/^.*upstream$/d' \
        -e '/^.*fallthrough.*$/d' \
        -e '/^.*forward . \/etc\/resolv.conf$/d' \
        -e '/^.*loop$/d' \
    )
    echo "Patched CoreDNS config:"
    echo "${fixed_coredns}"
    printf '%s' "${fixed_coredns}" | kubectl apply -f -
  fi

  # ginkgo regexes
  SKIP="${SKIP:-}"
  FOCUS="${FOCUS:-"\\[Conformance\\]"}"
  # if we set PARALLEL=true, skip serial tests set --ginkgo-parallel
  if [ "${PARALLEL:-false}" = "true" ]; then
    export GINKGO_PARALLEL=y
    if [ -z "${SKIP}" ]; then
      SKIP="\\[Serial\\]"
    else
      SKIP="\\[Serial\\]|${SKIP}"
    fi
  fi

  # setting this env prevents ginkgo e2e from trying to run provider setup
  export KUBERNETES_CONFORMANCE_TEST='y'
  # setting these is required to make RuntimeClass tests work ...
  export KUBE_CONTAINER_RUNTIME=remote
  export KUBE_CONTAINER_RUNTIME_ENDPOINT=unix:///run/containerd/containerd.sock
  export KUBE_CONTAINER_RUNTIME_NAME=containerd
  # unless we specify otherwise, tolerate flakes :/
  # TODO: remove this when kubernetes e2e testing no longer tolerates flakes ...
  # https://github.com/kubernetes/test-infra/blob/608557bbe4342da9895a0762cbdbf8a508ee59e9/config/jobs/kubernetes/sig-cloud-provider/gcp/gcp-gce.yaml#L5-L6
  export GINKGO_TOLERATE_FLAKES="${GINKGO_TOLERATE_FLAKES:-y}"
  ./hack/ginkgo-e2e.sh \
    '--provider=skeleton' "--num-nodes=${NUM_NODES}" \
    "--ginkgo.focus=${FOCUS}" "--ginkgo.skip=${SKIP}" \
    "--report-dir=${ARTIFACTS}" '--disable-log-dump=true'
}

main() {
  # create temp dir and setup cleanup
  TMP_DIR=$(mktemp -d)
  trap cleanup EXIT

  # ensure artifacts (results) directory exists when not in CI
  export ARTIFACTS="${ARTIFACTS:-${PWD}/_artifacts}"
  mkdir -p "${ARTIFACTS}"

  # default to bazel
  # TODO(bentheelder): remove this line once we've updated CI to explicitly choose
  BUILD_TYPE="${BUILD_TYPE:-bazel}"

  # build kubernetes
  if [ "${BUILD_TYPE:-}" = "bazel" ]; then
    build_with_bazel
  else
    build
  fi

  # in CI attempt to release some memory after building
  if [ -n "${KUBETEST_IN_DOCKER:-}" ]; then
    sync || true
    echo 1 > /proc/sys/vm/drop_caches || true
  fi

  # create the cluster and run tests
  create_cluster && run_tests
}

main
