# Server Operations

Operations guide for running tunapi-server in production.

## Service Management

```bash
# Check status
systemctl status tunapi

# Restart
systemctl restart tunapi

# View logs
journalctl -u tunapi -f -n 50

# Verify process
ps aux | grep tunapi
ss -tlnp | grep 8443
```

## Files & Locations

| File | Description |
|---|---|
| Binary | `/usr/local/bin/tunapi` |
| Routes | `/etc/tunapi/routes.json` |
| Agents | `/etc/tunapi/agents.json` |
| Service | `/etc/systemd/system/tunapi.service` |
| nginx site | `/etc/nginx/sites-enabled/tunapps` |

## Environment Variables

| Variable | Default | Required |
|---|---|---|
| `TUNAPI_PORT` | `8443` | No |
| `TUNAPI_SECRET` | `changeme` | **Yes** |
| `TUNAPI_BASE_DOMAIN` | `tunapi.local` | **Yes** |
| `TUNAPI_PUBLIC_SCHEME` | `https` | No |
| `TUNAPI_ROUTES_FILE` | `/etc/tunapi/routes.json` | No |
| `TUNAPI_AGENTS_FILE` | `/etc/tunapi/agents.json` | No |
| `TUNAPI_ALLOWED_TARGETS` | `127.0.0.1,localhost` | No |

## Redeploy

```bash
# Build locally
cd ~/tunapi
go build -ldflags="-s -w" -o tunapi-server .

# Deploy
scp tunapi-server root@your-server:/tmp/tunapi
ssh root@your-server "systemctl stop tunapi && \
  cp /tmp/tunapi /usr/local/bin/tunapi && \
  systemctl start tunapi"
```

## Agent Persistence

Agents are persisted to `/etc/tunapi/agents.json`. If you recreate the agent store (format change), the file is rewritten on next save. Old agent records remain but with `online: false`.

## SSL / TLS

SSL is terminated at nginx. nginx proxies to `127.0.0.1:8443` (tunapi). See `tunapps.nginx.conf` for the nginx site template.
