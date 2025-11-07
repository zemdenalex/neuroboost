#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "[1/4] docker compose pull/build"
docker-compose build --pull

echo "[2/4] restart stack"
docker-compose down
docker-compose up -d

echo "[3/4] status"
docker-compose ps

echo "[4/4] tail API logs (press Ctrl+C to stop)"
docker-compose logs -f api
