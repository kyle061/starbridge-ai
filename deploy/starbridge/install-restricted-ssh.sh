#!/usr/bin/env bash
# Run once as a server administrator. The private deployment key never reaches the server.
set -euo pipefail
[[ $EUID == 0 ]] || { echo 'Run this installer as root.' >&2; exit 1; }
public_key_file=${1:?public key file required}
app_port=${2:-18080}
source_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
deploy_root=/opt/starbridge
deploy_home=/var/lib/starbridge-deploy

command -v sudo >/dev/null
command -v python3 >/dev/null
ssh-keygen -lf "$public_key_file" >/dev/null
[[ $(wc -l < "$public_key_file") == 1 ]] || { echo 'Provide exactly one public key.' >&2; exit 1; }
[[ $(cat "$public_key_file") == ssh-ed25519\ * ]] || { echo 'An Ed25519 public key is required.' >&2; exit 1; }
if id starbridge-deploy >/dev/null 2>&1; then
  [[ -f "$deploy_home/.starbridge-account" ]] || { echo 'Account already exists; refusing to change it.' >&2; exit 1; }
fi

bash "$source_dir/deploy-server.sh" prepare "$deploy_root" "$app_port"
install -d -m 755 "$deploy_root/deploy/starbridge" /usr/local/libexec
for name in compose.yaml .env.example init-env.py deploy-server.sh Caddyfile; do
  install -o root -g root -m 644 "$source_dir/$name" "$deploy_root/deploy/starbridge/$name"
done
install -o root -g root -m 755 "$source_dir/ssh-gateway.py" /usr/local/libexec/starbridge-ssh-gateway.py
install -o root -g root -m 755 "$source_dir/deploy-admin.py" /usr/local/sbin/starbridge-deploy-admin
if ! id starbridge-deploy >/dev/null 2>&1; then
  useradd --system --user-group --home-dir "$deploy_home" --no-create-home --shell /bin/bash starbridge-deploy
fi
install -d -o root -g root -m 755 "$deploy_home" "$deploy_home/.ssh"
install -d -o starbridge-deploy -g starbridge-deploy -m 700 "$deploy_home/incoming"
touch "$deploy_home/.starbridge-account"
printf 'command="/usr/bin/python3 -I /usr/local/libexec/starbridge-ssh-gateway.py",restrict %s\n' "$(cat "$public_key_file")" > "$deploy_home/.ssh/authorized_keys"
chown root:root "$deploy_home/.ssh/authorized_keys"
chmod 644 "$deploy_home/.ssh/authorized_keys"
sudoers_temp=$(mktemp)
trap 'rm -f -- "$sudoers_temp"' EXIT
printf '%s\n' 'Defaults:starbridge-deploy !requiretty' 'starbridge-deploy ALL=(root) NOPASSWD: /usr/local/sbin/starbridge-deploy-admin' > "$sudoers_temp"
visudo -cf "$sudoers_temp"
install -o root -g root -m 440 "$sudoers_temp" /etc/sudoers.d/starbridge-deploy
echo 'Restricted Starbridge deployment account installed. Existing SSH accounts and Docker projects were not modified.'
