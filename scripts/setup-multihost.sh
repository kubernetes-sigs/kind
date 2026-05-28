#!/usr/bin/env bash
# setup-multihost.sh — fait les étapes 2 et 3 du checklist :
#   - clé SSH ed25519 sur le manager (générée si absente)
#   - propage la clé publique sur chaque worker (via ssh, demandera le
#     mot de passe une fois par worker si SSH par mdp est activé)
#   - ajoute les host keys des workers dans known_hosts du manager
#   - crée le docker context pour chaque worker
#   - vérifie que chaque daemon répond
#
# À lancer DEPUIS le manager (ex: ecotype-5).
#
# Usage:
#   ./setup-multihost.sh <worker1> [<worker2> ...]
# Exemple:
#   ./setup-multihost.sh ecotype-6 ecotype-47

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "usage: $0 <worker1> [<worker2> ...]" >&2
    exit 2
fi
WORKERS=("$@")

step() { echo; echo "════════ $* ════════"; }

# ─── 1. clé SSH ──────────────────────────────────────────────────────
step "1. clé SSH locale"
if [ ! -f ~/.ssh/id_ed25519 ]; then
    ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
fi
PUB=$(cat ~/.ssh/id_ed25519.pub)
echo "clé publique : ${PUB:0:60}..."

# ─── 2. propager la clé + récupérer la host key ──────────────────────
step "2. propagation de la clé sur ${#WORKERS[@]} worker(s)"
mkdir -p ~/.ssh && chmod 700 ~/.ssh
touch ~/.ssh/known_hosts && chmod 600 ~/.ssh/known_hosts

NEEDS_MANUAL=()

for w in "${WORKERS[@]}"; do
    echo
    echo "── $w ──"
    # known_hosts (évite "Host key verification failed" sur les commandes ultérieures)
    ssh-keyscan -H "$w" >> ~/.ssh/known_hosts 2>/dev/null || true

    if ssh -o BatchMode=yes -o ConnectTimeout=5 "root@$w" true 2>/dev/null; then
        echo "$w : clé déjà installée ✓"
        continue
    fi

    # tentative d'install via ssh-copy-id (interactif mdp)
    if command -v ssh-copy-id >/dev/null && \
       ssh-copy-id -o StrictHostKeyChecking=accept-new "root@$w" 2>/dev/null; then
        echo "$w : clé installée via ssh-copy-id ✓"
        continue
    fi

    echo "$w : install auto impossible (SSH par mdp désactivé sur le worker)"
    NEEDS_MANUAL+=("$w")
done

# Si certains workers réclament une install manuelle, on imprime UNE FOIS
# en fin de section un bloc prêt à coller pour chaque worker concerné, puis
# on s'arrête.
if [ "${#NEEDS_MANUAL[@]}" -gt 0 ]; then
    cat >&2 <<EOF

════════ ACTION MANUELLE REQUISE ════════

Connecte-toi à chacun des workers ci-dessous (depuis ton frontend
Grid'5000, pas depuis ce manager) et lance EXACTEMENT ce bloc :

    mkdir -p ~/.ssh && chmod 700 ~/.ssh
    echo '$PUB' >> ~/.ssh/authorized_keys
    chmod 600 ~/.ssh/authorized_keys

Workers concernés :
EOF
    for w in "${NEEDS_MANUAL[@]}"; do
        echo "    ssh root@$w" >&2
    done
    cat >&2 <<EOF

Puis relance :
    $0 ${WORKERS[*]}

EOF
    exit 1
fi

# ─── 3. vérifier passwordless ────────────────────────────────────────
step "3. vérif passwordless"
for w in "${WORKERS[@]}"; do
    ssh -o BatchMode=yes -o ConnectTimeout=5 "root@$w" echo "$w OK"
done

# ─── 4. docker contexts ──────────────────────────────────────────────
step "4. docker contexts"
for w in "${WORKERS[@]}"; do
    if docker context inspect "$w" >/dev/null 2>&1; then
        echo "$w : context déjà créé ✓"
    else
        docker context create "$w" --docker host="ssh://root@$w"
    fi
done

# ─── 5. vérif daemons distants ───────────────────────────────────────
step "5. tous les daemons répondent ?"
for w in default "${WORKERS[@]}"; do
    if v=$(docker --context "$w" version --format '{{.Server.Version}}' 2>/dev/null); then
        echo "$w : docker $v ✓"
    else
        echo "$w : FAIL — le daemon ne répond pas" >&2
        exit 1
    fi
done

step "PRÊT — tu peux maintenant lancer ./bin/kind avec ton multi.yaml"
