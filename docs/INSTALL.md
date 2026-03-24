# Install

Production installation with systemd and nginx.

## Step 1: Build

```bash
git clone https://github.com/andyeswong/tunapi.git
cd tunapi
go build -ldflags="-s -w" -o tunapi .
go build -ldflags="-s -w" -o tunagent ./cmd/tunagent
go build -ldflags="-s -w" -o tunctl ./cmd/tunctl
```

## Step 2: Install Binary

```bash
sudo install -m 0755 tunapi /usr/local/bin/tunapi
sudo install -m 0755 tunctl /usr/local/bin/tunctl
```

## Step 3: Create Service User

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tunapi
sudo mkdir -p /etc/tunapi
sudo chown tunapi:tunapi /etc/tunapi
```

## Step 4: Environment File

`/etc/tunapi/tunapi.env`:

```bash
TUNAPI_SECRET=your-strong-secret-here
TUNAPI_BASE_DOMAIN=tunapps.example.com
TUNAPI_PUBLIC_SCHEME=https
TUNAPI_PORT=8443
TUNAPI_ROUTES_FILE=/etc/tunapi/routes.json
TUNAPI_AGENTS_FILE=/etc/tunapi/agents.json
TUNAPI_ALLOWED_TARGETS=127.0.0.1,localhost
```

## Step 5: systemd Unit

`/etc/systemd/system/tunapi.service`:

```ini
[Unit]
Description=TunAPI dynamic reverse-proxy
After=network.target

[Service]
User=tunapi
Group=tunapi
EnvironmentFile=/etc/tunapi/tunapi.env
ExecStart=/usr/local/bin/tunapi
Restart=always
RestartSec=2
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/tunapi

[Install]
WantedBy=multi-user.target
```

## Step 6: Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now tunapi
sudo systemctl status tunapi
```

## Step 7: nginx Frontend

Use `tunapps.nginx.conf` as a template. Key points:
- Listen on `:80` for ACME challenges → proxy to port `9180`
- Listen on `:443` with SSL → proxy to `127.0.0.1:8443`
- WebSocket support: `Upgrade` and `Connection` headers

```bash
sudo cp tunapps.nginx.conf /etc/nginx/sites-available/tunapi
sudo ln -s /etc/nginx/sites-available/tunapi /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```
