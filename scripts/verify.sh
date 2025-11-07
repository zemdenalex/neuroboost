#!/usr/bin/env bash
set -euo pipefail

echo "[check] docker ps"
docker-compose ps

echo "[check] API health (container)"
curl -fsS http://127.0.0.1:8080/api/health || { echo "API health failed"; exit 1; }

echo "[check] Nginx → API (host)"
curl -fsS https://neuroboost.website/api/health || { echo "Nginx/API public check failed"; exit 1; }

echo "[ok] All checks passed"
