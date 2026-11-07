#!/usr/bin/env bash
# deploy.sh — the `deploy` and `deploy-dev` jobs of .github/workflows/ci.yml, run
# from this machine over ssh.
#
#   scripts/deploy.sh dev  [--no-e2e]
#   scripts/deploy.sh prod
#
# What it refuses (each of these is a step GitHub used to enforce for free):
#   - a commit scripts/ci-local.sh has not passed (the stamp in .git/ci-local.ok)
#   - a commit that is not on the remote yet: the host does `git pull origin
#     <branch>` from GitHub, so what it builds is whatever GitHub has
#   - dev from a branch other than develop, prod from a branch other than main
#   - prod without a human at the keyboard typing the confirmation. The merge to
#     main is the release and needs Denis's own word (CLAUDE.md); a script that
#     can be run with no tty is a script an agent can run, so this one cannot.
#
# Host: root@62.76.228.106 (ssh key from ~/.ssh, BatchMode). Prod /opt/neuroboost,
# staging /root/neuroboost-dev.
#
# Not copied from CI: the Telegram «deploy ok/failed» message. The terminal this
# runs in is where the answer is read; add it back (bot token + chat id from the
# project's local memory dir, never the repo) if it is missed.
set -euo pipefail

cd "$(dirname "$0")/.."
TARGET="${1:-}"
NO_E2E=0
[ "${2:-}" = "--no-e2e" ] && NO_E2E=1

HOST="${DEPLOY_HOST:-root@62.76.228.106}"
SSH=(ssh -o BatchMode=yes -o ConnectTimeout=15 "$HOST")

case "$TARGET" in
  dev)  BRANCH=develop; URL=https://dev.neuroboost.space ;;
  prod) BRANCH=main;    URL=https://neuroboost.space ;;
  *) echo "usage: scripts/deploy.sh dev [--no-e2e] | prod" >&2; exit 2 ;;
esac

die() { echo "deploy: $*" >&2; exit 1; }

[ "$(git rev-parse --abbrev-ref HEAD)" = "$BRANCH" ] || die "$TARGET deploys $BRANCH; you are on $(git rev-parse --abbrev-ref HEAD)"
git fetch -q origin "$BRANCH" || die "cannot reach GitHub to check what the host will pull"
HEAD_SHA="$(git rev-parse HEAD)"
[ "$HEAD_SHA" = "$(git rev-parse "origin/$BRANCH")" ] || die "HEAD is not what origin/$BRANCH has: push first (the host pulls from GitHub)"

STAMPED="$(cut -d' ' -f1 .git/ci-local.ok 2>/dev/null || true)"
if [ "$STAMPED" != "$HEAD_SHA" ]; then
  # A commit after the stamp is fine if it only touched documentation.
  changed="$(git diff --name-only "${STAMPED:-$HEAD_SHA}" "$HEAD_SHA" 2>/dev/null || echo UNKNOWN)"
  if [ -z "$STAMPED" ] || [ -z "$changed" ] || printf '%s\n' "$changed" | grep -qvE '^(docs/|graph/|.*\.md$)'; then
    die "${HEAD_SHA:0:7} has no green scripts/ci-local.sh (stamped: ${STAMPED:+${STAMPED:0:7}}${STAMPED:-none}). Run it first."
  fi
fi

echo "deploy $TARGET: ${HEAD_SHA:0:7} ($BRANCH) -> $HOST"

if [ "$TARGET" = "prod" ]; then
  [ -t 0 ] && [ -t 1 ] || die "prod needs a person at a terminal: no tty here"
  echo "This is the release. The merge to main needs Denis's own word."
  printf "Type 'deploy prod' to continue: "
  read -r answer
  [ "$answer" = "deploy prod" ] || die "not confirmed"
fi

remote_common='
set -e
retry() { for i in 1 2 3; do "$@" && return 0; echo "attempt $i failed, retrying in 5s..."; sleep 5; done; return 1; }
'

if [ "$TARGET" = "prod" ]; then
  # 🔴 A failed backup STOPS the deploy (ci.yml: it was once swallowed for four
  # months). This deploy applies migrations; no verified backup, no deploy.
  "${SSH[@]}" "bash -s" <<REMOTE
$remote_common
cd /opt/neuroboost
git status --short | grep -v '^?? .env.bak' && { echo "host has uncommitted changes; git pull would refuse"; exit 1; } || true
bash ./scripts/backup.sh
retry git pull origin main
docker compose build --pull
docker compose down
docker compose up -d
sleep 15
curl -fsS http://127.0.0.1:8080/api/health || exit 1
echo "Deploy successful"
REMOTE
else
  "${SSH[@]}" "bash -s" <<REMOTE
$remote_common
cd /root/neuroboost-dev
retry git fetch origin develop
git checkout develop
retry git pull origin develop
docker compose -f docker-compose.dev.yml build --pull
docker compose -f docker-compose.dev.yml down --remove-orphans
docker compose -f docker-compose.dev.yml up -d
sleep 15
curl -fsS http://127.0.0.1:8081/api/health || exit 1
echo "Dev deploy successful"
REMOTE
fi

echo "public check: $URL/api/health"
curl -fsS -m 20 "$URL/api/health" >/dev/null || die "the host says it is up, the public URL does not answer"
echo "  200"

if [ "$TARGET" = "dev" ] && [ "$NO_E2E" = 0 ]; then
  echo "e2e against staging (web/scripts/e2e-local.sh --staging)"
  bash web/scripts/e2e-local.sh --staging
fi

echo "done: $TARGET is on ${HEAD_SHA:0:7}"
if [ "$TARGET" = "prod" ]; then
  echo "Not done by this script (as in CI): the prod bot on nl-2 — see docs/relizy/<version>.md section 3."
fi
