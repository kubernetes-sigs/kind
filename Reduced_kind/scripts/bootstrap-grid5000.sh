#!/bin/bash
# bootstrap-grid5000.sh — from-zero multi-host Reduced_kind setup, N hosts.
#
# Captures every command we ran by hand during the Grid'5000 multi-host
# bring-up, generalised to N hosts (manager + N-1 workers).
#
# Usage (run as root on the MANAGER host):
#
#   ./bootstrap-grid5000.sh                       # auto-detect from $OAR_NODE_FILE
#   ./bootstrap-grid5000.sh <h1> <h2> ...         # explicit list, h1 = manager
#
# Examples:
#   ./bootstrap-grid5000.sh                            # in an OAR session
#   ./bootstrap-grid5000.sh ecotype-5 ecotype-12 ecotype-48
#
# The script is idempotent — re-running just re-checks each step.

set -euo pipefail

# ─── Phase 0 · figure out the host list ────────────────────────────────
if [ $# -eq 0 ]; then
    if [ -z "${OAR_NODE_FILE:-}" ] || [ ! -r "${OAR_NODE_FILE}" ]; then
        cat <<EOF
usage:
   $0                              (auto-detect: needs \$OAR_NODE_FILE)
   $0 <manager> <worker1> ...      (explicit short hostnames)
EOF
        exit 2
    fi
    mapfile -t HOSTS < <(cat "$OAR_NODE_FILE" | awk -F. '{print $1}' | uniq)
else
    HOSTS=("$@")
fi

if [ "${#HOSTS[@]}" -lt 1 ]; then
    echo "no hosts found" >&2; exit 2
fi

MANAGER="${HOSTS[0]}"
WORKERS=("${HOSTS[@]:1}")

# Site name in the FQDN.  All Grid'5000 hosts in one OAR job are on one
# site; pick from the first hostname's full form.
SITE="${SITE:-nantes}"
fqdn() { echo "$1.${SITE}.grid5000.fr"; }

REPO_URL="${REPO_URL:-https://github.com/Clement-NI/kind_extension_for_arena.git}"
REPO_BRANCH="${REPO_BRANCH:-claude/implement-kind-create-cluster-6ixKk}"
REPO_DIR="${REPO_DIR:-$HOME/kind_extension_for_arena}"

step() { echo; echo "════════ $* ════════"; }

step "topology: manager=$MANAGER workers=(${WORKERS[*]})  site=$SITE"

# ─── Phase 1 · install Docker on every host (parallel) ─────────────────
step "Phase 1 · install Docker on all ${#HOSTS[@]} hosts (parallel)"
install_docker() {
    local h="$1"
    if [ "$h" = "$MANAGER" ]; then
        bash -s <<'INSTALL'
set -e
if ! command -v docker >/dev/null; then
    apt-get update
    apt-get install -y docker.io || {
        sed -i 's|^deb http://security.debian.org|# &|' /etc/apt/sources.list
        apt-get update
        apt-get install -y docker.io
    }
fi
systemctl enable --now docker
INSTALL
    else
        ssh "$h" 'bash -s' <<'INSTALL'
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
INSTALL
    fi
}

for h in "${HOSTS[@]}"; do
    ( install_docker "$h" && echo "   docker OK on $h" ) &
done
wait

# ─── Phase 2 · git + Go on manager ─────────────────────────────────────
step "Phase 2 · install git + Go on manager"
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

# ─── Phase 3 · passwordless SSH manager → every worker ─────────────────
step "Phase 3 · passwordless SSH manager → workers"
[ -f ~/.ssh/id_ed25519 ] || ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
PUBKEY="$(cat ~/.ssh/id_ed25519.pub)"

for w in "${WORKERS[@]}"; do
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$w" true 2>/dev/null; then
        echo "   $w: passwordless SSH already works"
    else
        echo "   $w: installing pubkey (may prompt once)"
        ssh "$w" "
            mkdir -p ~/.ssh && chmod 700 ~/.ssh
            grep -qxF '$PUBKEY' ~/.ssh/authorized_keys 2>/dev/null \
                || echo '$PUBKEY' >> ~/.ssh/authorized_keys
            chmod 600 ~/.ssh/authorized_keys
        "
    fi
    # accept FQDN host key so 'docker --context' works later
    ssh-keyscan -H "$(fqdn "$w")" 2>/dev/null >> ~/.ssh/known_hosts
done
sort -u ~/.ssh/known_hosts -o ~/.ssh/known_hosts

# ─── Phase 4 · docker context for every worker ─────────────────────────
step "Phase 4 · docker context for each worker"
for w in "${WORKERS[@]}"; do
    if ! docker context inspect "$w" >/dev/null 2>&1; then
        docker context create "$w" \
            --docker "host=ssh://root@$(fqdn "$w")"
        echo "   created context $w"
    else
        echo "   context $w already exists"
    fi
    docker --context="$w" version >/dev/null
done

# ─── Phase 5 · clone + build reducedkind ───────────────────────────────
step "Phase 5 · clone + build reducedkind"
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

# ─── Phase 6 · gather IPs and run multi-host test ──────────────────────
step "Phase 6 · gather internal IPs"
MGR_IP="$(hostname -I | awk '{print $1}')"
echo "   manager $MANAGER → $MGR_IP"
WORKER_SPECS=()
for w in "${WORKERS[@]}"; do
    ip="$(ssh -o BatchMode=yes "$w" "hostname -I | awk '{print \$1}'")"
    WORKER_SPECS+=("${w}=${ip}")
    echo "   worker  $w → $ip"
done

step "Phase 7 · run end-to-end multi-host test"
"$REPO_DIR/Reduced_kind/scripts/test-multihost.sh" \
    "$MGR_IP" "${WORKER_SPECS[@]}" demo

step "DONE"
echo
echo "Cluster 'demo' is up across:"
echo "  CP     $MANAGER ($MGR_IP)"
for spec in "${WORKER_SPECS[@]}"; do
    echo "  worker $(echo "$spec" | awk -F= '{print $1 " (" $2 ")"}')"
done
echo
echo "Tear down:"
HOSTS_ARG="default=$MGR_IP"
for spec in "${WORKER_SPECS[@]}"; do
    HOSTS_ARG="$HOSTS_ARG,$spec"
done
echo "  $REPO_DIR/bin/reducedkind --multihost --hosts \"$HOSTS_ARG\" delete demo"
echo "  docker swarm leave --force"
for w in "${WORKERS[@]}"; do
    echo "  ssh $w docker swarm leave --force"
done
