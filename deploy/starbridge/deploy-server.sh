#!/usr/bin/env bash
# Used over SSH by deploy.yml. All resources belong to this dedicated installation.
set -euo pipefail

mode=${1:?prepare or apply required}
deploy_root=${2:?dedicated directory required}
app_port=${3:?application port required}
marker_text='https://github.com/kyle061/starbridge-ai'

fail() { printf '%s\n' "$*" >&2; exit 1; }
[[ "$mode" == prepare || "$mode" == apply ]] || fail 'Unknown deployment operation.'
[[ "$deploy_root" =~ ^/[a-zA-Z0-9_./-]+/starbridge$ ]] || fail 'Directory must be absolute and end in /starbridge.'
[[ "$deploy_root" != *'/../'* && "$deploy_root" != *'/./'* && "$deploy_root" != *'//'* ]] || fail 'Directory must not contain traversal components.'
[[ "$app_port" =~ ^[1-9][0-9]{0,4}$ ]] || fail 'Invalid application port.'
(( app_port >= 1024 && app_port <= 65535 )) || fail 'Application port must be between 1024 and 65535.'
for command in docker python3 curl; do
  command -v "$command" >/dev/null || fail "Install $command before deploying."
done
docker info >/dev/null
docker compose version >/dev/null
[[ "$(python3 -c 'import pathlib,sys; print(pathlib.Path(sys.argv[1]).resolve())' "$deploy_root")" == "$deploy_root" ]] || fail 'Deployment directory must not resolve through a symlink.'

if [[ "$mode" == prepare ]]; then
  if [[ ! -f "$deploy_root/.starbridge-managed" ]]; then
    if [[ -e "$deploy_root" ]] && [[ -n "$(find "$deploy_root" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
      fail 'Target directory already contains files; choose a new dedicated directory.'
    fi
    [[ -z "$(docker ps -aq --filter label=com.docker.compose.project=starbridge)" ]] || fail 'A different starbridge Compose project already exists.'
    [[ -z "$(docker volume ls -q --filter name=starbridge_)" ]] || fail 'Existing starbridge volumes require manual review before first deployment.'
    [[ -z "$(docker network ls -q --filter name=starbridge_)" ]] || fail 'An existing starbridge network requires manual review before first deployment.'
    mkdir -p "$deploy_root"
    printf '%s\n' "$marker_text" > "$deploy_root/.starbridge-managed"
  fi
fi
[[ -f "$deploy_root/.starbridge-managed" && ! -L "$deploy_root/.starbridge-managed" ]] || fail 'Managed installation marker is missing.'
[[ "$(cat "$deploy_root/.starbridge-managed")" == "$marker_text" ]] || fail 'Installation marker does not match Starbridge.'

# Verify both project ownership and port availability before loading/restarting anything.
python3 - "$deploy_root" "$app_port" <<'PY'
import json
import socket
import subprocess
import sys

root, port = sys.argv[1], int(sys.argv[2])
ids = subprocess.check_output([
    "docker", "ps", "-aq", "--filter", "label=com.docker.compose.project=starbridge"
], text=True).split()
owns_port = False
if ids:
    for container in json.loads(subprocess.check_output(["docker", "inspect", *ids])):
        labels = container["Config"].get("Labels") or {}
        if labels.get("com.docker.compose.project.working_dir") != root + "/deploy/starbridge":
            sys.exit("Existing starbridge container belongs to another directory; stopping deployment.")
        if labels.get("com.docker.compose.service") == "gateway" and container["State"]["Running"]:
            bindings = container["NetworkSettings"].get("Ports", {}).get("8080/tcp") or []
            owns_port = any(int(binding["HostPort"]) == port for binding in bindings)
if not owns_port:
    try:
        with socket.socket() as sock:
            sock.bind(("0.0.0.0", port))
    except OSError:
        sys.exit(f"Port {port} is already occupied; choose another port. No existing service was stopped.")
PY

if [[ "$mode" == prepare ]]; then
  mkdir -p "$deploy_root/deploy/starbridge"
  case "$(uname -m)" in
    x86_64) echo linux/amd64 ;;
    aarch64|arm64) echo linux/arm64 ;;
    *) fail 'Only amd64 and arm64 servers are supported.' ;;
  esac
  exit 0
fi

admin_email=${4:?administrator email required}
image=${5:?revision image required}
[[ "$image" =~ ^starbridge-ai:[a-f0-9]{40}$ ]] || fail 'Image must use a Starbridge commit revision tag.'
cd "$deploy_root/deploy/starbridge"
[[ ! -L .env ]] || fail 'The environment file must not be a symlink.'
if [[ ! -f .env ]]; then
  python3 init-env.py --admin-email "$admin_email"
fi
python3 - "$app_port" "$image" <<'PY'
import os
from pathlib import Path
import sys
import tempfile

path = Path(".env")
updates = {"APP_PORT": sys.argv[1], "BIND_HOST": "0.0.0.0", "STARBRIDGE_IMAGE": sys.argv[2]}
lines = []
for line in path.read_text().splitlines():
    key = line.partition("=")[0]
    lines.append(f"{key}={updates.pop(key)}" if key in updates else line)
lines.extend(f"{key}={value}" for key, value in updates.items())
fd, temporary = tempfile.mkstemp(prefix=".env-", dir=".")
with os.fdopen(fd, "w") as output:
    output.write("\n".join(lines) + "\n")
os.replace(temporary, path)
PY
docker load --input "$deploy_root/starbridge-image.tar.gz"
compose=(docker compose -p starbridge --project-directory "$PWD" -f "$PWD/compose.yaml" --env-file "$PWD/.env")
"${compose[@]}" config --quiet
if ! "${compose[@]}" up -d --no-build --pull missing --wait --wait-timeout 240 gateway postgres redis; then
  "${compose[@]}" ps
  fail 'Starbridge did not become healthy; check its container logs. Existing projects were not stopped.'
fi
curl --fail --silent --show-error --max-time 10 "http://127.0.0.1:$app_port/health" >/dev/null
"${compose[@]}" ps
rm -- "$deploy_root/starbridge-image.tar.gz"
