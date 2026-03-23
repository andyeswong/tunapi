# TunAPI

Minimal dynamic subdomain reverse-proxy registry.

## Documentation
- Quickstart: [GETTING_STARTED.md](./GETTING_STARTED.md)
- Install: [INSTALL.md](./INSTALL.md)
- User Manual: [USER_MANUAL.md](./USER_MANUAL.md)
- API: [API.md](./API.md)
- CLI Admin: [CLI.md](./CLI.md)
- Troubleshooting: [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)

## Features
- Register routes dynamically (`POST /register`)
- Delete routes (`DELETE /register?subdomain=...`)
- List routes (`GET /list`)
- Proxy requests by subdomain to target host:port
- Route persistence in JSON file

## Security defaults (important)
- `TUNAPI_SECRET` is mandatory (service fails if left as `changeme`)
- `/list` requires secret auth
- Target host must be in `TUNAPI_ALLOWED_TARGETS`
- HTTP server and reverse proxy use sane timeouts

## Environment variables
Copy `.env.example` and set values:

- `TUNAPI_SECRET` **required**
- `TUNAPI_BASE_DOMAIN` **required** (e.g. `tunapps.example.com`)
- `TUNAPI_PUBLIC_SCHEME` (`http` or `https`)
- `TUNAPI_PORT` (default `8443`)
- `TUNAPI_ROUTES_FILE` (default `/etc/tunapi/routes.json`)
- `TUNAPI_ALLOWED_TARGETS` (CSV allowlist)

## Run
```bash
go build -o tunapi .
TUNAPI_SECRET='super-secret' \
TUNAPI_BASE_DOMAIN='tunapps.example.com' \
TUNAPI_PUBLIC_SCHEME='https' \
TUNAPI_ALLOWED_TARGETS='127.0.0.1,localhost' \
./tunapi
```

## API

### Health
```bash
curl -sS http://127.0.0.1:8443/health
```

### Register route
```bash
curl -sS -X POST 'http://127.0.0.1:8443/register' \
  -H 'X-Secret: super-secret' \
  -H 'Content-Type: application/json' \
  -d '{"subdomain":"app","target":"127.0.0.1","port":8080}'
```

### List routes
```bash
curl -sS 'http://127.0.0.1:8443/list' -H 'X-Secret: super-secret'
```

### Delete route
```bash
curl -sS -X DELETE 'http://127.0.0.1:8443/register?subdomain=app' \
  -H 'X-Secret: super-secret'
```

## Reverse proxy behavior
Incoming host must match `*.TUNAPI_BASE_DOMAIN`. Example:
- Host: `app.tunapps.example.com`
- TunAPI looks up route `app`
- Proxies to configured `target:port`

## Notes for public repo
- Do not commit real secrets
- Do not commit private deployment configs with internal IPs/domains
- Keep `TUNAPI_ALLOWED_TARGETS` strict in production
