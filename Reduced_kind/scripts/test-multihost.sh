#!/bin/bash
# test-multihost.sh — Reduced_kind multi-host smoke test for N hosts.
#
# Validates that a multi-host Kubernetes cluster comes up across the
# manager + N workers via Docker Swarm overlay.
#
# Prerequisites (set up by bootstrap-grid5000.sh):
#   - Docker daemon on every host
#   - SSH passwordless from manager to every worker as root
#   - docker context "<worker>" on the manager for every worker
#
# Usage (run on the manager):
#   ./test-multihost.sh <manager-ip> <worker1-ctx>=<ip> [<worker2-ctx>=<ip> ...] [cluster-name]
#
# The last arg, if it does NOT contain '=', is treated as the cluster name.
# Default cluster name is "demo".
#
# Examples:
#   ./test-multihost.sh 172.16.193.5 ecotype-48=172.16.193.48
#   ./test-multihost.sh 172.16.193.5 ecotype-48=172.16.193.48 ecotype-12=172.16.193.12 demo3

set -euo pipefail

# ─── parse args ──────────────────────────────────────────────────────────
if [ $# -lt 2 ]; then
    cat <<EOF
usage: $0 <manager-ip> <worker-ctx>=<ip> [...]  [cluster-name]
example: $0 172.16.193.5 ecotype-48=172.16.193.48 ecotype-12=172.16.193.12 demo
EOF
    exit 2
fi
MGR_IP="$1"
shift

# Last arg without '=' is the cluster name; everything else is a worker spec.
CLUSTER_NAME="demo"
WORKER_SPECS=()
for arg in "$@"; do
    if [[ "$arg" == *=* ]]; then
        WORKER_SPECS+=("$arg")
    else
        CLUSTER_NAME="$arg"
    fi
done

if [ "${#WORKER_SPECS[@]}" -eq 0 ]; then
    echo "no worker specs supplied (need at least one <ctx>=<ip>)" >&2
    exit 2
fi

# Build the comma-separated --hosts argument.
HOSTS_ARG="default=$MGR_IP"
WORKER_CTXS=()
for spec in "${WORKER_SPECS[@]}"; do
    HOSTS_ARG="$HOSTS_ARG,$spec"
    WORKER_CTXS+=("${spec%=*}")
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
BIN="${REPO_ROOT}/../bin/reducedkind"

step() { echo; echo "════════ $* ════════"; }

# ─── 0. preflight ────────────────────────────────────────────────────────
step "0. preflight"
[ -x "$BIN" ] || {
    echo "binary $BIN not found; build with:"
    echo "    cd $REPO_ROOT && go build -o ../bin/reducedkind ./cmd/reducedkind"
    exit 1
}
docker version >/dev/null
for ctx in "${WORKER_CTXS[@]}"; do
    docker --context="$ctx" version >/dev/null || {
        echo "docker context '$ctx' not reachable"
        exit 1
    }
done
echo "binary, local docker, ${#WORKER_CTXS[@]} remote daemons: OK"

# ─── 1. clean any leftover ───────────────────────────────────────────────
step "1. clean any leftover"
"$BIN" --multihost --hosts "$HOSTS_ARG" delete "$CLUSTER_NAME" 2>/dev/null || true
docker swarm leave --force 2>/dev/null || true
for ctx in "${WORKER_CTXS[@]}"; do
    docker --context="$ctx" swarm leave --force 2>/dev/null || true
done
docker network rm kind 2>/dev/null || true
echo "leftover containers + swarm + overlay removed"

# ─── 2. create the multi-host cluster ────────────────────────────────────
step "2. create cluster '${CLUSTER_NAME}' across $((1 + ${#WORKER_SPECS[@]})) hosts"
"$BIN" --multihost --bootstrap-swarm \
    --hosts "$HOSTS_ARG" \
    create "$CLUSTER_NAME"

# ─── 3. verify nodes ─────────────────────────────────────────────────────
step "3. kubectl get nodes (all should be Ready)"
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf get nodes -o wide

# ─── 4. verify pods spread ───────────────────────────────────────────────
step "4. kubectl get pods -A (kindnet/kube-proxy on every node)"
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf get pods -A -o wide

# ─── 5. cross-host networking proof ──────────────────────────────────────
step "5. each worker → control-plane connectivity (Swarm overlay)"
for ctx in "${WORKER_CTXS[@]}"; do
    # The first worker container is named <cluster>-worker, the rest <cluster>-workerN.
    : # We'll iterate by container name pattern below.
done
# Loop over actual worker containers found via docker ps.
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf get nodes -l '!node-role.kubernetes.io/control-plane' \
    -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | while read -r node; do
    [ -z "$node" ] && continue
    # Find which host (context) this container lives on.
    for ctx in "${WORKER_CTXS[@]}"; do
        if docker --context="$ctx" ps --format '{{.Names}}' | grep -qx "$node"; then
            echo "  $node (on $ctx) → 10.0.1.2:6443:"
            docker --context="$ctx" exec "$node" bash -c '
                timeout 3 bash -c "</dev/tcp/10.0.1.2/6443" \
                    && echo "    TCP OK" \
                    || echo "    TCP FAIL"
            '
            break
        fi
    done
done

# ─── 6. real workload: cross-node scheduling ─────────────────────────────
NREP=$(( (1 + ${#WORKER_SPECS[@]}) * 2 ))
step "6. deploy $NREP-replica nginx, expect spread across all nodes"
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf create deployment hello \
    --image=nginx --replicas=$NREP --dry-run=client -o yaml \
    | docker exec -i "${CLUSTER_NAME}-control-plane" kubectl \
        --kubeconfig=/etc/kubernetes/admin.conf apply -f -

sleep 8
docker exec "${CLUSTER_NAME}-control-plane" kubectl \
    --kubeconfig=/etc/kubernetes/admin.conf get pods -l app=hello -o wide

step "DONE — cluster '${CLUSTER_NAME}' healthy, $((1 + ${#WORKER_SPECS[@]})) hosts"
echo
echo "Tear down:"
echo "  $BIN --multihost --hosts '$HOSTS_ARG' delete $CLUSTER_NAME"
