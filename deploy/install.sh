#!/usr/bin/env bash
#
# Install/update go-nginx-proxy on the server.
# Run from the git clone on the server:
#
#     git pull && ./deploy/install.sh
#
# Safe to re-run (idempotent). Requires root/sudo for the install steps.
#
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE_NAME="go-nginx-proxy"
ENV_FILE="/etc/go-nginx-proxy.env"
BIN_DIR="/usr/local/bin"

cd "$REPO_DIR"

echo "==> building frontend"
if [ ! -d web/node_modules ]; then
  (cd web && npm install --no-audit --no-fund)
fi
(cd web && npm run build)
touch internal/web/dist/.gitkeep

echo "==> building binary"
CGO_ENABLED=0 go build -o bin/go-nginx-proxy ./cmd/server

echo "==> installing binary to ${BIN_DIR}/go-nginx-proxy"
sudo install -m 0755 bin/go-nginx-proxy "${BIN_DIR}/go-nginx-proxy"

echo "==> ensuring env file"
if [ ! -f "$ENV_FILE" ]; then
  sudo cp deploy/go-nginx-proxy.example.env "$ENV_FILE"
  sudo chmod 600 "$ENV_FILE"
  sudo chown root:root "$ENV_FILE"
  echo
  echo "    >>> EDIT $ENV_FILE and set PROXY_PASSWORD, then restart:"
  echo "        sudo systemctl restart $SERVICE_NAME"
  echo
fi

echo "==> installing systemd unit"
sudo cp deploy/go-nginx-proxy.service "/etc/systemd/system/${SERVICE_NAME}.service"
sudo systemctl daemon-reload
sudo systemctl enable "$SERVICE_NAME"

echo "==> installing nginx log format (proxy_logged, \$host first)"
sudo mkdir -p /etc/nginx/conf.d
sudo cp deploy/nginx-proxy-log.conf /etc/nginx/conf.d/nginx-proxy-log.conf
# point the default access log at the format (idempotent)
if ! grep -q 'proxy_logged' /etc/nginx/nginx.conf; then
  sudo sed -i 's|access_log /var/log/nginx/access.log;|access_log /var/log/nginx/access.log proxy_logged;|' /etc/nginx/nginx.conf
fi
if sudo nginx -t >/dev/null 2>&1; then
  sudo systemctl reload nginx
  echo "==> nginx reloaded with proxy_logged format"
else
  echo "!! nginx -t failed; reverting log format change" >&2
  sudo sed -i 's|access_log /var/log/nginx/access.log proxy_logged;|access_log /var/log/nginx/access.log;|' /etc/nginx/nginx.conf
  sudo rm -f /etc/nginx/conf.d/nginx-proxy-log.conf
fi

echo "==> restarting service"
sudo systemctl restart "$SERVICE_NAME"
sudo systemctl --no-pager --lines=20 status "$SERVICE_NAME"

echo
echo "==> install complete"
