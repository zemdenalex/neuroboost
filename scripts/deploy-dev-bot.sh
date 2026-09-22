#!/usr/bin/env bash
# deploy-dev-bot.sh — put a git ref of bot/ onto @NeuroBoost_dev_bot (nl-2).
#
#   scripts/deploy-dev-bot.sh [ref]      # default: develop
#
# CI does not deploy either bot (gotcha 19): «green tests» and «on the bot you
# can press» are different claims, and the second one needed this command typed
# by hand every time. This is that command, plus the check that makes it true:
# the script fails unless the new container logs «Bot started» after the build.
#
# DEV ONLY on purpose. The prod bot (/opt/neuroboost-bot-prod) is a release step
# and goes out only on Denis's «да» — not from a script that is one argument away.
#
# Leaves nl-2's neighbours alone: it touches only /opt/neuroboost-bot and the
# neuroboost-dev-bot container (the Nivium exit node lives on the same host).
# Every log line goes through the token filter (gotcha 14). --force-recreate:
# an unchanged image would otherwise keep the old container, log nothing new,
# and fail the «Bot started» check; it also re-reads .env (gotcha 13).
set -euo pipefail

REF="${1:-develop}"
KEY=~/.ssh/ufo_servers
HOST=root@185.214.10.107
DIR=/opt/neuroboost-bot
CONTAINER=neuroboost-dev-bot

cd "$(dirname "$0")/.."
redact() { sed -E 's/bot[0-9]+:[A-Za-z0-9_-]+/bot<REDACTED>/g'; }

SHA="$(git rev-parse --short "$REF^{commit}")" || { echo "❌ no such ref: $REF" >&2; exit 2; }
if [ "$REF" = develop ] && [ "$(git rev-parse develop)" != "$(git rev-parse origin/develop 2>/dev/null || true)" ]; then
  echo "⚠ local develop ($SHA) is not origin/develop — deploying what is local, not what CI tested"
fi

STAMP="$(date +%Y%m%d-%H%M%S)"
echo "▶ $REF ($SHA) → $HOST:$DIR"

# Keep the previous source as a rollback; only the three newest backups stay.
ssh -i $KEY $HOST "cd $DIR && mv src src.bak-$STAMP && ls -d src.bak-* | sort | head -n -3 | xargs -r rm -rf"
git archive --prefix=src/ "$REF:bot" | ssh -i $KEY $HOST "cd $DIR && tar x"
ssh -i $KEY $HOST "test -f $DIR/src/go.mod" || { echo "❌ src/ did not arrive" >&2; exit 1; }

STARTED="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ssh -i $KEY $HOST "cd $DIR && docker compose up -d --build --force-recreate bot" 2>&1 | tail -2

for _ in $(seq 1 20); do
  if ssh -i $KEY $HOST "docker logs --since $STARTED $CONTAINER 2>&1" | grep -q "Bot started"; then
    echo "🟢 $CONTAINER is up on $SHA"
    ssh -i $KEY $HOST "docker logs --since $STARTED $CONTAINER 2>&1" | redact | tail -4
    exit 0
  fi
  sleep 2
done

echo "❌ no «Bot started» within 40 s — last log lines:" >&2
ssh -i $KEY $HOST "docker logs --since $STARTED $CONTAINER 2>&1" | redact | tail -15 >&2
echo "   rollback: ssh … 'cd $DIR && rm -rf src && mv src.bak-$STAMP src && docker compose up -d --build bot'" >&2
exit 1
