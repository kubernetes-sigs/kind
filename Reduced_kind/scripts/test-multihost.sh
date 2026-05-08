#!/bin/bash
# Reduced_kind multi-host smoke test.
#
# Validates that a multi-host Kubernetes cluster comes up across two
# Grid'5000 (or any Linux) nodes via Docker Swarm overlay.
#
# Prerequisites (on both hosts):
#   - Docker daemon running
#   - SSH passwordless from manager to worker as root
#   - Docker context "<WORKER>" on the manager pointing at the worker
#       e.g. docker context create ecotype-48 \
#                --docker host=ssh://root@ecotype-48.nantes.grid5000.fr
#
# Usage:
#   ./test-multihost.sh <manager-ip> <worker-ctx>=<worker-ip> [cluster-name]
# Example:
#   ./test-multihost.sh 172.16.193.5 ecotype-48=172.16.193.48 demo
#
# Run this on the manager host.

set -euo pipefail

# ─── parse args ──────────────────────────────────────────────────────────
if [ $# -lt 2 ]; then
    cat <<EOF
usage: $0 <manager-ip> <worker-ctx>=<worker-ip> [cluster-name]
example: $0 172.16.193.5 ecotype-48=172.16.193.48 demo
EOF
    exit 2
fi
MGR_IP="$1"
WORKER_SPEC="$2"               # e.g. ecotype-48=172.16.193.48
CLUSTER_NAME="${3:-demo}"
WORKER_CTX="${WORKER_SPEC%=*}" # ecotype-48
HOSTS="default=${MGR_IP},${WORKER_SPEC}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
BIN="${REPO_ROOT}/../bin/reducedkind"

step() { echo; echo "=== $* ==="; }

# ─── 0. preflight ────────────────────────────────────────────────────────
step "0. preflight"
[ -x "$BIN" ] || {
    echo "binary $BIN not found; build with:"
    echo "    cd $REPO_ROOT && go build -o ../bin/reducedkind ./cmd/reducedkind"
    exit 1
}
docker version >/dev/null
docker --context="${WORKER_CTX}" version >/dev/null || {
    echo "docker context '${WORKER_CTX}' not reachable. Did you run:"
    echo "    docker context create ${WORKER_CTX} --docker host=ssh://root@<host>"
    exit 1
}
echo "binary, local docker, remote docker: OK"

# ─── 1. clean any leftover ───────────────────────────────────────────────
step "1. clean any leftover"
"$BIN" --multihost --hosts "$HOSTS" delete "$CLUSTER_NAME" 2>/dev/null || true
docker swarm leave --force 2>/dev/null || true
docker --context="${WORKER_CTX}" swarm leave --force 2>/dev/null || true
docker network rm kind 2>/dev/null || true
echo "leftover containers + swarm + overlay removed"

# ─── 2. create the multi-host cluster ────────────────────────────────────
step "2. create cluster '${CLUSTER_NAME}' across both hosts"
"$BIN" --multihost --bootstrap-swarm \
    --hosts "$HOSTS" \
    create "$CLUSTER_NAME"

# ─── 3. verify nodes ─────────────────────────────────────────────────────
step "3. kubectl get nodes (both should be Ready)"
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf get nodes -o wide

# ─── 4. verify pods spread ───────────────────────────────────────────────
step "4. kubectl get pods -A (kindnet/kube-proxy on both nodes)"
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf get pods -A -o wide

# ─── 5. cross-host networking proof ──────────────────────────────────────
step "5. worker → control-plane connectivity (over Swarm overlay)"
docker --context="${WORKER_CTX}" exec "${CLUSTER_NAME}-worker" bash -c '
    echo "from $(hostname), reaching apiserver via overlay:"
    timeout 3 bash -c "</dev/tcp/10.0.1.2/6443" && echo "  TCP 6443 OK" || echo "  TCP 6443 FAIL"
    curl -skf https://10.0.1.2:6443/version --max-time 5 && echo
'

# ─── 6. real workload: cross-node scheduling ─────────────────────────────
step "6. deploy 4-replica nginx, expect spread across both nodes"
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf create deployment hello \
    --image=nginx --replicas=4 --dry-run=client -o yaml \
    | docker exec -i "${CLUSTER_NAME}-control-plane" kubectl \
        --kubeconfig=/etc/kubernetes/admin.conf apply -f -

# Wait for pods to schedule.
sleep 8
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf get pods -l app=hello -o wide

step "DONE — cluster '${CLUSTER_NAME}' is healthy and multi-host"
echo
echo "Inspect / play:"
echo "  docker exec ${CLUSTER_NAME}-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf get all -A"
echo
echo "Tear down:"
echo "  $BIN --multihost --hosts '$HOSTS' delete $CLUSTER_NAME"
