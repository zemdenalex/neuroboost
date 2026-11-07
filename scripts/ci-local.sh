#!/usr/bin/env bash
# ci-local.sh — what .github/workflows/ci.yml runs in `backend`, `frontend` and
# `docker`, on this machine, in one command.
#
#   scripts/ci-local.sh [--fast] [--only backend,bot,frontend,docker]
#
# On success of ALL FOUR stages, with a clean tracked tree, it writes
# .git/ci-local.ok = "<HEAD sha> <time>". scripts/hooks/pre-push and
# scripts/deploy.sh refuse a commit that has no such stamp. --fast and --only
# skip stages and therefore never stamp.
#
# Why it is not just «run the tests» (graph: test-database memory): the Go tests
# that touch the database SKIP when DATABASE_URL is unset and the package still
# prints `ok`, and Go hands back a cached result for a database it does not count
# as an input. So: a throwaway Postgres is always started, DATABASE_URL always
# set, `-count=1` always passed, and the run FAILS if any test printed the
# «DATABASE_URL not set» skip. A green that did not run is the failure this
# script exists to refuse.
#
# Needs: docker (running), go, migrate (golang-migrate with the postgres driver),
# pnpm 10.12.0, node 22, curl. Ports used: 15433 (Postgres), 18080 (API).
set -uo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"
LOG_DIR="$ROOT/.git/ci-local"
mkdir -p "$LOG_DIR"

PG=nbci-pg
API=nbci-api
NET=nbci-net
# 🔴 Outside 49152–65535, the Windows dynamic range. 55433 sat inside it: any app
# (Telegram Desktop, 27.10) may hold it as an outgoing port, and then Docker
# publishes it without error while nothing on the host can reach it — «actively
# refused», or via localhost/::1 «forbidden by its access permissions».
PG_PORT=15433
API_PORT=18080
# 127.0.0.1, not localhost: the port is published on IPv4 only, and Go tries ::1
# first and reports THAT error, which hides the real one.
DSN="postgres://test:test@127.0.0.1:${PG_PORT}/neuroboost_test?sslmode=disable"
JWT="ci-local-jwt-secret-not-a-secret-0123456789abcdef"

STAGES="backend bot frontend docker"
FULL=1
while [ $# -gt 0 ]; do
  case "$1" in
    --fast) STAGES="backend bot frontend"; FULL=0 ;;
    --only) STAGES="${2//,/ }"; FULL=0; shift ;;
    -h|--help) sed -n 2,25p "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
done

cleanup() {
  docker rm -f "$API" >/dev/null 2>&1
  docker rm -f "$PG" >/dev/null 2>&1
  docker network rm "$NET" >/dev/null 2>&1
}
trap cleanup EXIT

declare -A RESULT TIME
FAILED=0

run_stage() {
  local name="$1" start end
  echo
  echo "━━ $name ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  start=$(date +%s)
  if "stage_$name" > >(tee "$LOG_DIR/$name.log") 2>&1; then
    RESULT[$name]="PASS"
  else
    RESULT[$name]="FAIL"
    FAILED=1
  fi
  wait
  end=$(date +%s)
  TIME[$name]=$((end - start))
}

need_db() {
  # One throwaway Postgres for backend and docker. Re-created each run: no state
  # from last time can make a test pass.
  docker info >/dev/null 2>&1 || { echo "docker is not running (Docker Desktop does not start itself after a reboot)"; return 1; }
  if docker ps --format '{{.Names}}' | grep -qx "$PG"; then return 0; fi
  cleanup
  docker network create "$NET" >/dev/null || return 1
  docker run -d --name "$PG" --network "$NET" --network-alias db \
    -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=neuroboost_test \
    -p "127.0.0.1:${PG_PORT}:5432" postgres:16-alpine >/dev/null || return 1
  local i
  for i in $(seq 1 40); do
    docker exec "$PG" pg_isready -U test -d neuroboost_test >/dev/null 2>&1 && break
    sleep 1
  done
  # pg_isready answers during init, before the final restart: ask a real query.
  for i in $(seq 1 40); do
    docker exec "$PG" psql -U test -d neuroboost_test -tAc 'select 1' >/dev/null 2>&1 && break
    sleep 1
  done
  [ "$i" -lt 40 ] || docker exec "$PG" psql -U test -d neuroboost_test -tAc 'select 1' >/dev/null 2>&1 \
    || { echo "Postgres did not come up"; return 1; }
  # Ready inside the container is not proof the host can reach it (see PG_PORT):
  # wait for the host side, so a dead forward fails here with its real name.
  for i in $(seq 1 30); do
    (exec 3<>"/dev/tcp/127.0.0.1/${PG_PORT}") 2>/dev/null && return 0
    sleep 1
  done
  echo "Postgres did not come up on 127.0.0.1:${PG_PORT}"; return 1
}

stage_backend() {
  need_db || return 1
  ( cd api-go && migrate -path migrations -database "$DSN" up ) || return 1
  ( cd api-go && go build -o bin/api ./cmd/api && go build -o bin/healthcheck ./cmd/healthcheck ) || return 1
  local out="$LOG_DIR/go-test-api.txt"
  ( cd api-go && DATABASE_URL="$DSN" JWT_SECRET="$JWT" go test -count=1 -v ./... ) > "$out" 2>&1
  local code=$?
  grep -E "^(FAIL|ok |--- FAIL|panic)" "$out" | tail -40
  if grep -q "DATABASE_URL not set" "$out"; then
    echo "REFUSED: DB-backed tests were skipped even though DATABASE_URL is set; a green here would prove nothing"
    return 1
  fi
  local ran skipped
  ran=$(grep -c "^--- PASS\|^    --- PASS" "$out" || true)
  skipped=$(grep -c "^--- SKIP\|^    --- SKIP" "$out" || true)
  echo "tests passed: $ran, skipped: $skipped (skips are listed in $out)"
  return $code
}

stage_bot() {
  # bot/ is a separate Go module: api-go's build never compiles it (ci.yml says so).
  ( cd bot && go build ./... && go vet ./... && go test -count=1 ./... ) 2>&1 | tail -30
  return "${PIPESTATUS[0]}"
}

stage_frontend() {
  # CI=true makes pnpm freeze the lockfile, as on GitHub's runner.
  ( cd web && CI=true pnpm install --frozen-lockfile && pnpm typecheck && pnpm lint && pnpm test --run && pnpm build ) 2>&1 | tail -40
  return "${PIPESTATUS[0]}"
}

stage_docker() {
  need_db || return 1
  # Full output kept in a file, its tail shown on failure: with -q a failed
  # `pnpm install` inside the image said only «exit code: 1» (25.10).
  build() {
    local what="$1"; shift
    docker build "$@" > "$LOG_DIR/docker-build-$what.txt" 2>&1 || {
      echo "$what image failed to build; last lines:"; tail -25 "$LOG_DIR/docker-build-$what.txt"; return 1; }
  }
  build api -f api-go/Dockerfile -t nbci-api . || return 1
  build web --build-arg VITE_API_URL=http://localhost/api -t nbci-web ./web || return 1
  build bot -t nbci-bot ./bot || return 1
  docker rm -f "$API" >/dev/null 2>&1
  docker run -d --name "$API" --network "$NET" -p "127.0.0.1:${API_PORT}:8080" \
    -e "DATABASE_URL=postgres://test:test@db:5432/neuroboost_test?sslmode=disable" \
    -e "JWT_SECRET=$JWT" -e NODE_ENV=test -e API_PORT=8080 \
    -e TELEGRAM_BOT_TOKEN=fake:token -e TELEGRAM_NOTIFICATION_BOT_TOKEN=fake:token \
    -e FRONTEND_URL=http://localhost nbci-api >/dev/null || return 1
  local i
  for i in $(seq 1 30); do
    if curl -sf "http://127.0.0.1:${API_PORT}/api/health" >/dev/null 2>&1; then
      echo "API image answers /api/health"
      return 0
    fi
    sleep 2
  done
  echo "=== API logs ==="; docker logs "$API" 2>&1 | tail -30
  return 1
}

echo "ci-local: $(git rev-parse --short HEAD) on $(git rev-parse --abbrev-ref HEAD) · stages: $STAGES"
for s in $STAGES; do
  case "$s" in
    backend|bot|frontend) run_stage "$s" ;;
    docker)
      if [ "$FAILED" = 1 ]; then RESULT[docker]="SKIP"; TIME[docker]=0; else run_stage docker; fi ;;
    *) echo "unknown stage: $s" >&2; exit 2 ;;
  esac
done

echo
echo "━━ result ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for s in $STAGES; do printf "  %-9s %-5s %4ss   (log: .git/ci-local/%s.log)\n" "$s" "${RESULT[$s]:-?}" "${TIME[$s]:-0}" "$s"; done

if [ "$FAILED" = 1 ]; then
  echo "RED — nothing stamped."
  exit 1
fi

if [ "$FULL" != 1 ]; then
  echo "GREEN, but only some stages ran (--fast / --only): no stamp, push and deploy will still ask for a full run."
  exit 0
fi

# docs/ and graph/ are not under test, and always have somebody's notes in them.
if ! git diff --quiet -- . ':(exclude)docs' ':(exclude)graph' || ! git diff --cached --quiet -- . ':(exclude)docs' ':(exclude)graph'; then
  echo "GREEN, but tracked code has uncommitted changes: the result describes the working tree, not a commit. No stamp. Commit, then run again."
  exit 0
fi

echo "$(git rev-parse HEAD) $(date -Iseconds)" > "$ROOT/.git/ci-local.ok"
echo "GREEN — stamped $(git rev-parse --short HEAD). Push and deploy will accept this commit."
