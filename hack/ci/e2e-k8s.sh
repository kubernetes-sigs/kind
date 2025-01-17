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

set -o errexit -o nounset -o xtrace

SKIP="${SKIP:-}"
FOCUS="${FOCUS:-}"
LABEL_FILTER="${LABEL_FILTER:-}"
GA_ONLY="${GA_ONLY:-false}"
FEATURE_GATES="${FEATURE_GATES:-{\}}"
RUNTIME_CONFIG="${RUNTIME_CONFIG:-{\}}"
KIND_CLUSTER_LOG_LEVEL="${KIND_CLUSTER_LOG_LEVEL:-4}"
CLUSTER_LOG_FORMAT="${CLUSTER_LOG_FORMAT:-}"
KUBELET_LOG_FORMAT="${KUBELET_LOG_FORMAT:-${CLUSTER_LOG_FORMAT}}"
IP_FAMILY="${IP_FAMILY:-ipv4}"
KUBE_PROXY_MODE="${KUBE_PROXY_MODE:-iptables}"
PARALLEL="${PARALLEL:-false}"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

cleanup() {
    if [ "${CLEANED_UP:-false}" = "true" ]; then
        return
    fi
    if [ "${KIND_CREATE_ATTEMPTED:-false}" = "true" ]; then
        kind "export" logs "${ARTIFACTS}" || true
        kind delete cluster || true
    fi
    rm -f _output/bin/e2e.test || true
    if [ -n "${TMP_DIR:-}" ]; then
        rm -rf "${TMP_DIR:?}"
    fi
    CLEANED_UP=true
}

signal_handler() {
    if [ -n "${GINKGO_PID:-}" ]; then
        kill -TERM "${GINKGO_PID}" || true
    fi
    cleanup
}

build() {
    kind build node-image -v 1
    GINKGO_SRC_DIR="vendor/github.com/onsi/ginkgo/v2/ginkgo"
    if [ ! -d "${GINKGO_SRC_DIR}" ]; then
        GINKGO_SRC_DIR="vendor/github.com/onsi/ginkgo/ginkgo"
    fi
    make all WHAT="cmd/kubectl test/e2e/e2e.test ${GINKGO_SRC_DIR}"
    export PATH="${PWD}/_output/bin:${PATH}"
}

check_structured_log_support() {
    case "${KUBE_VERSION}" in
        v1.1[0-8].*)
            log "$1 is only supported on versions >= v1.19, got ${KUBE_VERSION}"
            exit 1
            ;;
    esac
}

create_cluster() {
    KUBE_VERSION="$(docker run --rm --entrypoint=cat "kindest/node:latest" /kind/version)"
    
    # Create cluster configuration
    cat <<EOF > "${ARTIFACTS}/kind-config.yaml"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  ipFamily: ${IP_FAMILY}
  kubeProxyMode: ${KUBE_PROXY_MODE}
  dnsSearch: []
nodes:
- role: control-plane
- role: worker
- role: worker
featureGates: ${FEATURE_GATES}
runtimeConfig: ${RUNTIME_CONFIG}
kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  metadata:
    name: config
  apiServer:
    extraArgs:
      v: "${KIND_CLUSTER_LOG_LEVEL}"
  controllerManager:
    extraArgs:
      v: "${KIND_CLUSTER_LOG_LEVEL}"
  scheduler:
    extraArgs:
      v: "${KIND_CLUSTER_LOG_LEVEL}"
  ---
  kind: InitConfiguration
  nodeRegistration:
    kubeletExtraArgs:
      v: "${KIND_CLUSTER_LOG_LEVEL}"
      container-log-max-files: "10"
      container-log-max-size: "100Mi"
  ---
  kind: JoinConfiguration
  nodeRegistration:
    kubeletExtraArgs:
      v: "${KIND_CLUSTER_LOG_LEVEL}"
      container-log-max-files: "10"
      container-log-max-size: "100Mi"
EOF

    # Create cluster
    KIND_CREATE_ATTEMPTED=true
    kind create cluster \
        --image=kindest/node:latest \
        --retain \
        --wait=1m \
        -v=3 \
        "--config=${ARTIFACTS}/kind-config.yaml"

    # Patch kube-proxy
    kubectl patch -n kube-system daemonset/kube-proxy \
        --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/command/-", "value": "--v='"${KIND_CLUSTER_LOG_LEVEL}"'" }]'
}

run_tests() {
    if [ "${IP_FAMILY}" = "ipv6" ]; then
        # Patch CoreDNS configuration
        original_coredns=$(kubectl get -oyaml -n=kube-system configmap/coredns)
        fixed_coredns=$(
            printf '%s' "${original_coredns}" | sed \
                -e 's/^.*kubernetes cluster\.local/& internal/' \
                -e '/^.*upstream$/d' \
                -e '/^.*fallthrough.*$/d' \
                -e '/^.*forward . \/etc\/resolv.conf$/d' \
                -e '/^.*loop$/d'
        )
        printf '%s' "${fixed_coredns}" | kubectl apply -f -
    fi

    # Configure Ginkgo
    if [ -z "${FOCUS}" ] && [ -z "${LABEL_FILTER}" ]; then
        FOCUS="\\[Conformance\\]"
    fi
    if [ "${PARALLEL}" = "true" ]; then
        export GINKGO_PARALLEL=y
        SKIP="${SKIP:+${SKIP}|}\\[Serial\\]"
    fi

    # Set environment variables
    export KUBERNETES_CONFORMANCE_TEST='y'
    export KUBE_CONTAINER_RUNTIME=remote
    export KUBE_CONTAINER_RUNTIME_ENDPOINT=unix:///run/containerd/containerd.sock
    export KUBE_CONTAINER_RUNTIME_NAME=containerd

    # Run tests
    ./hack/ginkgo-e2e.sh \
        '--provider=skeleton' "--num-nodes=2" \
        "--ginkgo.focus=${FOCUS}" "--ginkgo.skip=${SKIP}" "--ginkgo.label-filter=${LABEL_FILTER}" \
        "--report-dir=${ARTIFACTS}" '--disable-log-dump=true' &
    GINKGO_PID=$!
    wait "${GINKGO_PID}"
    if ! wait "${GINKGO_PID}"; then
    log "Ginkgo tests failed"
    exit 1
    fi

}

main() {
    TMP_DIR=$(mktemp -d)
    export ARTIFACTS="${ARTIFACTS:-${PWD}/_artifacts}"
    mkdir -p "${ARTIFACTS}"
    export KUBECONFIG="${HOME}/.kube/kind-test-config"
    
    trap signal_handler INT TERM

    log "Starting e2e tests"
    build
    create_cluster
    run_tests
    cleanup
}

main
