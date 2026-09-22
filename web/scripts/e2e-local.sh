#!/usr/bin/env bash
# e2e-local.sh — run Playwright specs from this machine in one command.
#
#   web/scripts/e2e-local.sh [--staging] [--project desktop|mobile] [spec ...]
#
# Default: THIS working tree's web app (vite on localhost:5173) against the
# staging API — the way to see a fix go green before it is pushed.
# --staging: no local build, specs run against https://dev.neuroboost.website,
# i.e. what CI's e2e job sees — the way to see a defect red before fixing it.
#
# Why a script (22.09): doing this by hand took ten tool calls, and every one of
# them was a trap that looked like a test failure:
#   - vite must run from the REAL path C:\E_Drive\…, not E:\ (gotcha 1);
#   - staging's CORS allows exactly http://localhost:5173 — 127.0.0.1 fails
#     preflight, so vite binds 127.0.0.1 but the browser is sent to localhost;
#   - E2E_TG_BOT_TOKEN is the DEV bot's token, read from nl-2 over ssh and never
#     printed; every line of output goes through the token filter (gotcha 14);
#   - E2E_TG_ID is personal data and lives outside git, in the project's local
#     memory dir (e2e-tg-id.txt), or in $E2E_TG_ID.
# ⚠ In a Claude Code session this needs the sandbox off: the sandbox refuses
#   loopback connections, and a refused connection reads as «vite is down».
set -euo pipefail

STAGING=0
PROJECT_ARGS=()
SPECS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --staging) STAGING=1 ;;
    --project) PROJECT_ARGS=(--project "$2"); shift ;;
    *) SPECS+=("$1") ;;
  esac
  shift
done

# The real path: E: is a subst drive for C:\E_Drive.
WEB="$(cd "$(dirname "$0")/.." && pwd)"
WEB="${WEB/#\/e\//\/c\/E_Drive\/}"
WEB="${WEB/#\/E\//\/c\/E_Drive\/}"
cd "$WEB"

redact() { sed -E 's/bot[0-9]+:[A-Za-z0-9_-]+/bot<REDACTED>/g'; }

ID_FILE="$HOME/.claude/projects/E--Projects-007---Ventures-V003---NeuroBoost/memory/e2e-tg-id.txt"
if [ -z "${E2E_TG_ID:-}" ]; then
  [ -s "$ID_FILE" ] || { echo "❌ no E2E_TG_ID and no $ID_FILE" >&2; exit 2; }
  E2E_TG_ID="$(tr -d '\r\n' < "$ID_FILE")"
fi
if [ -z "${E2E_TG_BOT_TOKEN:-}" ]; then
  E2E_TG_BOT_TOKEN="$(ssh -i ~/.ssh/ufo_servers root@185.214.10.107 \
    'grep "^TELEGRAM_BOT_TOKEN=" /opt/neuroboost-bot/.env | cut -d= -f2-' | tr -d '\r\n')"
  [ -n "$E2E_TG_BOT_TOKEN" ] || { echo "❌ could not read the dev bot token from nl-2" >&2; exit 2; }
fi
export E2E_TG_ID E2E_TG_BOT_TOKEN
export E2E_API_URL=https://dev.neuroboost.website

stop_vite() {
  # On Windows the pid bash holds is not vite's node process; kill whoever
  # listens on the port instead.
  for pid in $(netstat -ano 2>/dev/null | grep ':5173 ' | grep LISTENING | awk '{print $5}' | sort -u); do
    taskkill //PID "$pid" //F >/dev/null 2>&1 || true
  done
}

if [ "$STAGING" = 1 ]; then
  export E2E_BASE_URL=https://dev.neuroboost.website
  echo "▶ against staging (what CI sees)"
else
  if netstat -ano 2>/dev/null | grep ':5173 ' | grep -q LISTENING; then
    echo "❌ something already listens on :5173 — stop it first (it may be a build of another branch)" >&2
    exit 2
  fi
  trap stop_vite EXIT
  LOG="$(mktemp)"
  VITE_API_URL=https://dev.neuroboost.website/api \
    npx vite --host 127.0.0.1 --port 5173 --strictPort >"$LOG" 2>&1 &
  for _ in $(seq 1 30); do
    code="$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:5173/ || true)"
    [ "$code" = 200 ] && break
    sleep 1
  done
  if [ "${code:-}" != 200 ]; then
    echo "❌ vite did not answer on :5173 within 30 s (last code ${code:-none}); its log:" >&2
    tail -20 "$LOG" >&2
    exit 2
  fi
  export E2E_BASE_URL=http://localhost:5173
  echo "▶ local build of $(git rev-parse --short HEAD)$(git diff --quiet || echo '+dirty') against the staging API"
fi

npx playwright test "${SPECS[@]}" "${PROJECT_ARGS[@]}" --reporter=line 2>&1 | redact
exit "${PIPESTATUS[0]}"
