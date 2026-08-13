# go-nginx-proxy

A self-hosted **nginx reverse proxy manager** with a Go backend and a Svelte frontend. It manages nginx virtual host configs directly on the filesystem — **no database** — and integrates with **Let's Encrypt / certbot** including **Cloudflare DNS** verification.

## Features

- **Site management** — create/edit/delete nginx proxy configs as `domain.tld.conf` in `/etc/nginx/sites-available`
- **Enable/disable** — symlink configs into `/etc/nginx/sites-enabled`
- **nginx control** — start / stop / restart / reload and `nginx -t` config testing from the web UI
- **SSL certificates** — issue, renew and delete Let's Encrypt certs via certbot
  - `--dns-cloudflare` authenticator when a Cloudflare API token is configured
  - webroot fallback otherwise
- **Automatic reload** — nginx is reloaded after certificate issuance/renewal
- **Filesystem as source of truth** — each config embeds a `# meta:` JSON header so structured sites round-trip; external/raw configs are listed but flagged
- **Basic auth** — username/password from environment variables (no users stored in files)
- **Single binary** — the Svelte SPA is embedded into the Go binary

## Architecture

```
cmd/server          main entrypoint (HTTP server, static serving)
internal/api        REST handlers + basic auth
internal/nginx      config template + sites-available/enabled manager + service control
internal/certbot    certbot integration (cloudflare DNS / webroot)
internal/web        embedded Svelte SPA (//go:embed web/dist)
web/                Svelte 5 + Vite frontend
deploy/             systemd unit, container nginx config
```

## Quick start

```bash
make build            # builds web + Go binary -> bin/go-nginx-proxy
make run              # PROXY_ADDR=:8080 PROXY_USERNAME=admin PROXY_PASSWORD=changeme
```

Open http://localhost:8080 and log in with the configured credentials.

The server must run as **root** (or a user with write access to `/etc/nginx` and the certbot directories). If you prefer to run it as a non-root user, set `PROXY_SUDO=true` so commands are prefixed with `sudo` — then the process only needs passwordless sudo for the nginx/certbot commands.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `PROXY_ADDR` | `:8080` | HTTP listen address |
| `PROXY_USERNAME` | `admin` | Basic auth username |
| `PROXY_PASSWORD` | `changeme` | Basic auth password (leave empty to disable auth) |
| `PROXY_SUDO` | `false` | Prefix nginx/certbot commands with `sudo` |
| `PROXY_NGINX_AVAILABLE_DIR` | `/etc/nginx/sites-available` | Where `domain.conf` files are written |
| `PROXY_NGINX_ENABLED_DIR` | `/etc/nginx/sites-enabled` | Where symlinks are created |
| `PROXY_NGINX_BIN` | `nginx` | nginx binary |
| `PROXY_NGINX_CONFIG` | `/etc/nginx/nginx.conf` | Main config used for `nginx -t` |
| `PROXY_USE_SYSTEMCTL` | `true` | Use `systemctl` for service control (`false` uses the nginx binary directly) |
| `PROXY_SYSTEMCTL_BIN` | `systemctl` | systemctl binary |
| `PROXY_SERVICE_NAME` | `nginx` | systemd service name |
| `PROXY_CERTBOT_BIN` | `certbot` | certbot binary |
| `PROXY_CERTBOT_EMAIL` | *(empty)* | Certbot registration email |
| `PROXY_CERTBOT_STAGING` | `false` | Use Let's Encrypt staging CA |
| `PROXY_CERT_DIR` | `/etc/letsencrypt/live` | Where certbot stores certs |
| `PROXY_CLOUDFLARE_DNS_TOKEN` | *(empty)* | Cloudflare API token — enables `--dns-cloudflare` |
| `PROXY_CLOUDFLARE_CREDENTIALS_FILE` | `/etc/letsencrypt/cloudflare.ini` | Credentials file written from the token (chmod 600) |
| `PROXY_CLOUDFLARE_PROPAGATION` | `60` | DNS propagation wait (seconds) |
| `PROXY_WEB_DIR` | *(embedded)* | Serve the SPA from disk instead (frontend dev) |

## Frontend development

```bash
cd web
npm install
npm run dev        # Vite dev server proxying /api -> :8080
```

## API

All endpoints require Basic auth. JSON bodies use snake_case / camelCase per field.

```
GET    /api/health                         service health
GET    /api/nginx/status                   running state, version, config counts
POST   /api/nginx/start | stop | restart | reload
POST   /api/nginx/test                     nginx -t

GET    /api/sites                          list sites
POST   /api/sites                          create site
GET    /api/sites/{domain}                 get site
PUT    /api/sites/{domain}                 update site (domain change = rename)
DELETE /api/sites/{domain}                 delete site (and disable)
POST   /api/sites/{domain}/enable          create symlink in sites-enabled
POST   /api/sites/{domain}/disable         remove symlink
POST   /api/sites/{domain}/test            nginx -t (config must exist)

GET    /api/sites/{domain}/cert            certificate status
POST   /api/sites/{domain}/cert            issue certificate
POST   /api/sites/{domain}/cert/renew      renew certificate
DELETE /api/sites/{domain}/cert            delete certificate
```

Site object:

```json
{
  "domain": "example.com",
  "domains": ["www.example.com"],
  "ssl": true,
  "redirectToHttps": true,
  "clientMaxBodySize": "10m",
  "webRoot": "",
  "locations": [
    { "path": "/", "proxyPass": "http://127.0.0.1:3000", "websocket": true, "extra": "" }
  ],
  "extraConfig": "add_header X-Frame-Options SAMEORIGIN;"
}
```

## Generated config

Each site renders to `sites-available/example.com.conf`. For an SSL site with a cert issued:

```nginx
# managed-by: go-nginx-proxy
# meta: {...structured site...}

server {
    listen 80;
    listen [::]:80;
    server_name example.com www.example.com;

    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;
    server_name example.com www.example.com;
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Installation (Debian/Ubuntu)

```bash
make build
sudo make install                      # installs to /usr/local/bin
sudo cp deploy/go-nginx-proxy.service /etc/systemd/system/
sudo cp deploy/go-nginx-proxy.example.env /etc/go-nginx-proxy.env   # edit credentials
sudo systemctl daemon-reload && sudo systemctl enable --now go-nginx-proxy
```

## Security notes

- All endpoints (including the SPA) are behind HTTP Basic auth. Run behind TLS (e.g. via the proxy itself) or a trusted network.
- The service must be able to write to `/etc/nginx` and `/etc/letsencrypt` — do not expose it to the public internet.
- The Cloudflare API token is written to `PROXY_CLOUDFLARE_CREDENTIALS_FILE` with `0600` permissions.
- Configs are validated with `nginx -t` before reload where applicable; a bad config is never activated automatically.

## Roadmap

- Per-site raw config editing toggle
- Certificate auto-renew scheduling / status badge on the dashboard
- nginx log streaming
