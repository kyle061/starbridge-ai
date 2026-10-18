#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -f .env || { echo 'Missing private .env'; exit 1; }
docker network inspect starbridge_default >/dev/null
mkdir -p data
chown 1000:1000 data
chmod 700 data
docker compose -p liuying -f deploy/compose.yaml build
docker compose -p liuying -f deploy/compose.yaml up -d
for attempt in $(seq 1 40); do
  if curl --fail --silent http://127.0.0.1:18787/api/health; then
    echo
    echo 'Liuying container is ready. Validate/reload the Caddy route separately.'
    exit 0
  fi
  sleep 2
done
docker compose -p liuying -f deploy/compose.yaml logs --tail=40 liuying
exit 1
