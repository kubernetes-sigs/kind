#!/bin/bash
# bootstrap-grid5000.sh — complete from-zero multi-host Reduced_kind setup.
#
# Captures every command we ran by hand during the Grid'5000 multi-host
# bring-up: install Docker on both nodes, Go + git on the manager,
# passwordless SSH manager→worker, docker context, build, and finally
# run the multi-host smoke test.
#
# Usage (run as root on the MANAGER host):
#   ./bootstrap-grid5000.sh <manager-ip> <worker-shortname> <worker-ip>
# Example:
#   ./bootstrap-grid5000.sh 172.16.193.5 ecotype-48 172.16.193.48
#
# The script is idempotent — re-running just re-checks each step.

set -euo pipefail

# ─── args ──────────────────────────────────────────────────────────────
if [ $# -lt 3 ]; then
    cat <<EOF
usage: $0 <manager-ip> <worker-shortname> <worker-ip>
example: $0 172.16.193.5 ecotype-48 172.16.193.48
EOF
    exit 2
fi
MGR_IP="$1"
WORKER="$2"           # e.g. ecotype-48
WORKER_IP="$3"
WORKER_FQDN="${WORKER}.nantes.grid5000.fr"   # adjust site if not Nantes

REPO_URL="${REPO_URL:-https://github.com/Clement-NI/kind_extension_for_arena.git}"
REPO_BRANCH="${REPO_BRANCH:-claude/implement-kind-create-cluster-6ixKk}"
REPO_DIR="${REPO_DIR:-$HOME/kind_extension_for_arena}"

step() { echo; echo "════════ $* ════════"; }

# ─── Phase 1 · Docker on the MANAGER ───────────────────────────────────
step "Phase 1 · install Docker on manager (this host)"
if ! command -v docker >/dev/null; then
    apt-get update
    # Debian 11's stale security mirror sometimes 404s; keep going if so.
    apt-get install -y docker.io || {
        sed -i 's|^deb http://security.debian.org|# &|' /etc/apt/sources.list
        apt-get update
        apt-get install -y docker.io
    }
fi
systemctl enable --now docker
docker version | grep -A1 "^Server"

# ─── Phase 2 · Docker on the WORKER ────────────────────────────────────
step "Phase 2 · install Docker on worker ($WORKER) via SSH"
# At this point passwordless SSH may not be set up yet, so the user may
# need to enter a password once.
ssh "$WORKER" '
    set -e
    if ! command -v docker >/dev/null; then
        apt-get update
        apt-get install -y docker.io || {
            sed -i "s|^deb http://security.debian.org|# &|" /etc/apt/sources.list
            apt-get update
            apt-get install -y docker.io
        }
    fi
    systemctl enable --now docker
    docker version | grep -A1 "^Server"
'

# ─── Phase 3 · git + Go on manager ─────────────────────────────────────
step "Phase 3 · install git + Go on manager"
apt-get install -y git wget
if ! command -v go >/dev/null || [ "$(go version | awk '{print $3}')" \< "go1.22" ]; then
    cd /tmp
    wget -q https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
    grep -q '/usr/local/go/bin' ~/.bashrc || echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
fi
export PATH=/usr/local/go/bin:$PATH
go version

# ─── Phase 4 · passwordless SSH manager → worker ───────────────────────
step "Phase 4 · passwordless SSH manager → worker"
[ -f ~/.ssh/id_ed25519 ] || ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
PUBKEY="$(cat ~/.ssh/id_ed25519.pub)"

# Try to install our pubkey on the worker.  If passwordless SSH already
# works, this is a no-op; otherwise the user is prompted once.
if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$WORKER" true 2>/dev/null; then
    echo "(installing public key on $WORKER — you may be prompted once)"
    ssh "$WORKER" "
        mkdir -p ~/.ssh && chmod 700 ~/.ssh
        grep -qxF '$PUBKEY' ~/.ssh/authorized_keys 2>/dev/null \
            || echo '$PUBKEY' >> ~/.ssh/authorized_keys
        chmod 600 ~/.ssh/authorized_keys
    "
fi

# Pre-accept the FQDN host key so 'docker --context' won't fail later.
ssh-keyscan -H "$WORKER_FQDN" 2>/dev/null >> ~/.ssh/known_hosts
sort -u ~/.ssh/known_hosts -o ~/.ssh/known_hosts

ssh -o BatchMode=yes "$WORKER" hostname
ssh -o BatchMode=yes "$WORKER_FQDN" hostname

# ─── Phase 5 · docker context for the worker ───────────────────────────
step "Phase 5 · create docker context '$WORKER' on manager"
if ! docker context inspect "$WORKER" >/dev/null 2>&1; then
    docker context create "$WORKER" \
        --docker "host=ssh://root@${WORKER_FQDN}"
fi
docker --context="$WORKER" version | grep -A1 "^Server"

# ─── Phase 6 · clone + build reducedkind ───────────────────────────────
step "Phase 6 · clone + build reducedkind"
if [ ! -d "$REPO_DIR" ]; then
    git clone "$REPO_URL" "$REPO_DIR"
fi
cd "$REPO_DIR"
git fetch --all
git checkout "$REPO_BRANCH"
git pull --ff-only origin "$REPO_BRANCH"
cd Reduced_kind
go build -o ../bin/reducedkind ./cmd/reducedkind
cd ..
./bin/reducedkind 2>&1 | head -1 || true

# ─── Phase 7 · multi-host smoke test ───────────────────────────────────
step "Phase 7 · run end-to-end multi-host test"
"$REPO_DIR/Reduced_kind/scripts/test-multihost.sh" \
    "$MGR_IP" "${WORKER}=${WORKER_IP}" demo

step "DONE"
echo
echo "Cluster 'demo' is up across $MGR_IP (CP) and $WORKER_IP (worker)."
echo
echo "Tear down:"
echo "  $REPO_DIR/bin/reducedkind --multihost \\"
echo "      --hosts \"default=$MGR_IP,${WORKER}=$WORKER_IP\" delete demo"
echo "  docker swarm leave --force"
echo "  ssh $WORKER docker swarm leave --force"
