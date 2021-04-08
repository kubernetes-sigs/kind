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
# GA_ONLY: true  - limit to GA APIs/features as much as possible
#          false - (default) APIs and features left at defaults
# 

# cleanup logic for cleanup on exit
CLEANED_UP=false
cleanup() {
  if [ "$CLEANED_UP" = "true" ]; then
    return
  fi
  # KIND_CREATE_ATTEMPTED is true once we: kind create
  if [ "${KIND_CREATE_ATTEMPTED:-}" = true ]; then
    kind "export" logs "${ARTIFACTS}/logs" || true
    # get prometheus URL from the Service (servicePort = 8080)
    CLUSTER_IP=$(kubectl get svc prometheus-service -n monitoring -o jsonpath="{.spec.clusterIP}")
    # create a snapshot of the db
    docker exec kind-control-plane curl -XPOST http://"${CLUSTER_IP}":8080/api/v1/admin/tsdb/snapshot
    # get the prometheus database
    POD_NAME=$(kubectl -n monitoring get pods -o jsonpath='{.items[0].metadata.name}')
    kubectl -n monitoring exec "${POD_NAME}" -- tar cvf - /prometheus/snapshots > "${ARTIFACTS}/prometheus.tar"
    kind delete cluster || true
  fi
  rm -f _output/bin/e2e.test || true
  # remove our tempdir, this needs to be last, or it will prevent kind delete
  if [ -n "${TMP_DIR:-}" ]; then
    rm -rf "${TMP_DIR:?}"
  fi
  CLEANED_UP=true
}

# setup signal handlers
signal_handler() {
  if [ -n "${GINKGO_PID:-}" ]; then
    kill -TERM "$GINKGO_PID" || true
  fi
  cleanup
}
trap signal_handler INT TERM

# build kubernetes / node image, e2e binaries
build() {
  # build the node image w/ kubernetes
  kind build node-image -v 1
  # make sure we have e2e requirements
  make all WHAT='cmd/kubectl test/e2e/e2e.test vendor/github.com/onsi/ginkgo/ginkgo'
}

# up a cluster with kind
create_cluster() {
  # Grab the version of the cluster we're about to start
  KUBE_VERSION="$(docker run --rm --entrypoint=cat "kindest/node:latest" /kind/version)"

  # Default Log level for all components in test clusters
  KIND_CLUSTER_LOG_LEVEL=${KIND_CLUSTER_LOG_LEVEL:-4}

  # potentially enable --logging-format
  kubelet_extra_args="      \"v\": \"${KIND_CLUSTER_LOG_LEVEL}\""
  if [ -n "${KUBELET_LOG_FORMAT:-}" ]; then
    case "${KUBE_VERSION}" in
     v1.1[0-8].*)
      echo "KUBELET_LOG_FORMAT is only supported on versions >= v1.19, got ${KUBE_VERSION}"
      exit 1
      ;;
    *)
      # NOTE: the indendation on the next line is meaningful!
      kubelet_extra_args="${kubelet_extra_args}
      \"logging-format\": \"${KUBELET_LOG_FORMAT}\""
      ;;
    esac
  fi

  # JSON map injected into featureGates config
  feature_gates="{}"
  # --runtime-config argument value passed to the API server
  runtime_config="{}"

  case "${GA_ONLY:-false}" in
  false)
    feature_gates="{}"
    runtime_config="{}"
    ;;
  true)
    case "${KUBE_VERSION}" in
    v1.1[0-7].*)
      echo "GA_ONLY=true is only supported on versions >= v1.18, got ${KUBE_VERSION}"
      exit 1
      ;;
    v1.18.*)
      echo "Limiting to GA APIs and features (plus certificates.k8s.io/v1beta1 and RotateKubeletClientCertificate) for ${KUBE_VERSION}"
      feature_gates='{"AllAlpha":false,"AllBeta":false,"RotateKubeletClientCertificate":true}'
      runtime_config='{"api/alpha":"false", "api/beta":"false", "certificates.k8s.io/v1beta1":"true"}'
      ;;
    *)
      echo "Limiting to GA APIs and features for ${KUBE_VERSION}"
      feature_gates='{"AllAlpha":false,"AllBeta":false}'
      runtime_config='{"api/alpha":"false", "api/beta":"false"}'
      ;;
    esac
    ;;
  *)
    echo "\$GA_ONLY set to '${GA_ONLY}'; supported values are true and false (default)"
    exit 1
    ;;
  esac

  # create the config file
  cat <<EOF > "${ARTIFACTS}/kind-config.yaml"
# config for 1 control plane node and 2 workers (necessary for conformance)
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  ipFamily: ${IP_FAMILY:-ipv4}
  kubeProxyMode: ${KUBE_PROXY_MODE:-iptables}
nodes:
- role: control-plane
- role: worker
- role: worker
featureGates: ${feature_gates}
runtimeConfig: ${runtime_config}
kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  metadata:
    name: config
  apiServer:
    extraArgs:
      "v": "${KIND_CLUSTER_LOG_LEVEL}"
  controllerManager:
    extraArgs:
      "v": "${KIND_CLUSTER_LOG_LEVEL}"
      "bind-address": "0.0.0.0"
  scheduler:
    extraArgs:
      "v": "${KIND_CLUSTER_LOG_LEVEL}"
  ---
  kind: InitConfiguration
  nodeRegistration:
    kubeletExtraArgs:
${kubelet_extra_args}
  ---
  kind: JoinConfiguration
  nodeRegistration:
    kubeletExtraArgs:
${kubelet_extra_args}
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

  # Patch kube-proxy to set the verbosity level
  kubectl patch -n kube-system daemonset/kube-proxy \
    --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/command/-", "value": "--v='"${KIND_CLUSTER_LOG_LEVEL}"'" }]'
}

# install monitoring so we can get metrics
install_monitoring() {
  # Get the current config
  original_kube_proxy=$(kubectl get -oyaml -n=kube-system configmap/kube-proxy)
  echo "Original kube-proxy config:"
  echo "${original_kube_proxy}"
  # Patch it
  fixed_kube_proxy=$(
      printf '%s' "${original_kube_proxy}" | sed \
          's/\(.*metricsBindAddress:\)\( .*\)/\1 "0.0.0.0:10249"/' \
      )
  echo "Patched kube-proxy config:"
  echo "${fixed_kube_proxy}"
  printf '%s' "${fixed_kube_proxy}" | kubectl apply -f -
  # restart kube-proxy
  kubectl -n kube-system rollout restart ds kube-proxy

  cat <<'EOF' > "${ARTIFACTS}/kind-monitoring.yaml"
---
apiVersion: v1
kind: Namespace
metadata:
  name: monitoring
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus-service
  namespace: monitoring
  annotations:
      prometheus.io/scrape: 'true'
      prometheus.io/port:   '9090'
spec:
  selector: 
    app: prometheus-server
  type: NodePort
  ports:
    - port: 8080
      targetPort: 9090 
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: prometheus
rules:
- apiGroups: [""]
  resources:
  - nodes
  - nodes/proxy
  - services
  - endpoints
  - pods
  verbs: ["get", "list", "watch"]
- apiGroups:
  - extensions
  resources:
  - ingresses
  verbs: ["get", "list", "watch"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
- kind: ServiceAccount
  name: default
  namespace: monitoring
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-server-conf
  labels:
    name: prometheus-server-conf
  namespace: monitoring
data:
  prometheus.yml: |-
    global:
      scrape_interval: 5s
      evaluation_interval: 5s
    scrape_configs:
      - job_name: 'kubernetes-apiservers'
        kubernetes_sd_configs:
        - role: endpoints
        scheme: https
        tls_config:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
        relabel_configs:
        - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
          action: keep
          regex: default;kubernetes;https

      - job_name: 'kubernetes-controller-manager'
        honor_labels: true
        scheme: https
        tls_config:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
          insecure_skip_verify: true
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
        static_configs:
          - targets:
            - 127.0.0.1:10257

      - job_name: 'kubernetes-nodes'
        scheme: https
        tls_config:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
        kubernetes_sd_configs:
        - role: node
        relabel_configs:
        - action: labelmap
          regex: __meta_kubernetes_node_label_(.+)
        - target_label: __address__
          replacement: localhost:6443
        - source_labels: [__meta_kubernetes_node_name]
          regex: (.+)
          target_label: __metrics_path__
          replacement: /api/v1/nodes/${1}/proxy/metrics

      - job_name: 'kubernetes-cadvisor'
        scheme: https
        tls_config:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
        kubernetes_sd_configs:
        - role: node
        relabel_configs:
        - action: labelmap
          regex: __meta_kubernetes_node_label_(.+)
        - target_label: __address__
          replacement: localhost:6443
        - source_labels: [__meta_kubernetes_node_name]
          regex: (.+)
          target_label: __metrics_path__
          replacement: /api/v1/nodes/${1}/proxy/metrics/cadvisor

      - job_name: kube-proxy
        honor_labels: true
        kubernetes_sd_configs:
        - role: pod
        relabel_configs:
        - action: keep
          source_labels:
          - __meta_kubernetes_namespace
          - __meta_kubernetes_pod_name
          separator: '/'
          regex: 'kube-system/kube-proxy.+'
        - source_labels:
          - __address__
          action: replace
          target_label: __address__
          regex: (.+?)(\\:\\d+)?
          replacement: $1:10249
---          
apiVersion: v1
kind: Pod
metadata:
  name: prometheus
  namespace: monitoring
  labels:
    app: prometheus-server
spec:
  hostNetwork: true
  nodeSelector:
    node-role.kubernetes.io/master: ""
  tolerations:
  - key: CriticalAddonsOnly
    operator: Exists
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
  - effect: NoSchedule
    key: node-role.kubernetes.io/control-plane
  containers:
    - name: prometheus
      image: prom/prometheus:v2.26.0
      args:
        - "--config.file=/etc/prometheus/prometheus.yml"
        - "--storage.tsdb.path=/prometheus/"
        - "--web.enable-admin-api"
      ports:
        - containerPort: 9090
      volumeMounts:
        - name: prometheus-config-volume
          mountPath: /etc/prometheus/
        - name: prometheus-storage-volume
          mountPath: /prometheus/
  volumes:
    - name: prometheus-config-volume
      configMap:
        defaultMode: 420
        name: prometheus-server-conf
    - name: prometheus-storage-volume
      emptyDir: {}
EOF

  kubectl apply -f "${ARTIFACTS}/kind-monitoring.yaml"

}

# run e2es with ginkgo-e2e.sh
run_tests() {
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
  # setting these is required to make RuntimeClass tests work ... :/
  export KUBE_CONTAINER_RUNTIME=remote
  export KUBE_CONTAINER_RUNTIME_ENDPOINT=unix:///run/containerd/containerd.sock
  export KUBE_CONTAINER_RUNTIME_NAME=containerd
  # ginkgo can take forever to exit, so we run it in the background and save the
  # PID, bash will not run traps while waiting on a process, but it will while
  # running a builtin like `wait`, saving the PID also allows us to forward the
  # interrupt
  ./hack/ginkgo-e2e.sh \
    '--provider=skeleton' "--num-nodes=${NUM_NODES}" \
    "--ginkgo.focus=${FOCUS}" "--ginkgo.skip=${SKIP}" \
    "--report-dir=${ARTIFACTS}" '--disable-log-dump=true' &
  GINKGO_PID=$!
  wait "$GINKGO_PID"
}

main() {
  # create temp dir and setup cleanup
  TMP_DIR=$(mktemp -d)

  # ensure artifacts (results) directory exists when not in CI
  export ARTIFACTS="${ARTIFACTS:-${PWD}/_artifacts}"
  mkdir -p "${ARTIFACTS}"

  # export the KUBECONFIG to a unique path for testing
  KUBECONFIG="${HOME}/.kube/kind-test-config"
  export KUBECONFIG
  echo "exported KUBECONFIG=${KUBECONFIG}"

  # debug kind version
  kind version

  # build kubernetes
  build
  # in CI attempt to release some memory after building
  if [ -n "${KUBETEST_IN_DOCKER:-}" ]; then
    sync || true
    echo 1 > /proc/sys/vm/drop_caches || true
  fi

  # create the cluster and run tests
  res=0
  create_cluster || res=$?
  install_monitoring || res=$?
  run_tests || res=$?
  cleanup || res=$?
  exit $res
}

main
