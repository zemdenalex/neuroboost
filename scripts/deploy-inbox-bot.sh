#!/usr/bin/env bash
# deploy-inbox-bot.sh — put a git ref of bot/ onto the inbox bot (nl-2).
#
#   scripts/deploy-inbox-bot.sh [ref]      # default: develop
#
# First run needs /opt/neuroboost-inbox-bot/.env (deploy/inbox-bot/.env.example)
# — the token comes from Denis via BotFather and never passes through here.
# Which API the bot writes to is that .env's API_BASE, not this script.
# Fails unless the new container logs «Bot started». Touches only its own dir
# and container: nl-2 also runs both NeuroBoost bots and a Nivium exit node.
set -euo pipefail

REF="${1:-develop}"
KEY=~/.ssh/ufo_servers
HOST=root@185.214.10.107
DIR=/opt/neuroboost-inbox-bot
CONTAINER=neuroboost-inbox-bot

cd "$(dirname "$0")/.."
redact() { sed -E 's/bot[0-9]+:[A-Za-z0-9_-]+/bot<REDACTED>/g'; }

SHA="$(git rev-parse --short "$REF^{commit}")" || { echo "❌ no such ref: $REF" >&2; exit 2; }
ssh -i $KEY $HOST "test -f $DIR/.env" || { echo "❌ $HOST:$DIR/.env is missing — see deploy/inbox-bot/.env.example" >&2; exit 2; }
echo "▶ $REF ($SHA) → $HOST:$DIR"

STAMP="$(date +%Y%m%d-%H%M%S)"
scp -q -i $KEY deploy/inbox-bot/docker-compose.yml $HOST:$DIR/docker-compose.yml
ssh -i $KEY $HOST "cd $DIR && { [ -d src ] && mv src src.bak-$STAMP || true; } && ls -d src.bak-* 2>/dev/null | sort | head -n -3 | xargs -r rm -rf"
git archive --prefix=src/ "$REF:bot" | ssh -i $KEY $HOST "cd $DIR && tar x"
ssh -i $KEY $HOST "test -f $DIR/src/cmd/inbox/main.go" || { echo "❌ src/ did not arrive" >&2; exit 1; }

# The server's clock, not this PC's: docker logs --since compares against it.
STARTED="$(ssh -i $KEY $HOST date -u +%Y-%m-%dT%H:%M:%SZ)"
ssh -i $KEY $HOST "cd $DIR && docker compose up -d --build --force-recreate inbox" 2>&1 | tail -2

for _ in $(seq 1 20); do
  if ssh -i $KEY $HOST "docker logs --since $STARTED $CONTAINER 2>&1" | grep -q "Bot started"; then
    echo "🟢 $CONTAINER is up on $SHA"
    ssh -i $KEY $HOST "docker logs --since $STARTED $CONTAINER 2>&1" | redact | tail -3
    exit 0
  fi
  sleep 2
done
echo "❌ no «Bot started» within 40 s — last log lines:" >&2
ssh -i $KEY $HOST "docker logs --since $STARTED $CONTAINER 2>&1" | redact | tail -15 >&2
exit 1
