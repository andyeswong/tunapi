# Getting Started

5-minute guide to get TunAPI running.

## 1) Prerequisites

- Linux/macOS
- Go 1.21+
- A domain (or subdomain) pointing to your server's public IP
- nginx (recommended for SSL termination)

## 2) Build

```bash
git clone https://github.com/andyeswong/tunapi.git
cd tunapi
go build -o tunapi .
go build -o tunagent ./cmd/tunagent
go build -o tunctl ./cmd/tunctl
```

## 3) Minimal Configuration

```bash
export TUNAPI_SECRET='your-secret-here'
export TUNAPI_BASE_DOMAIN='tunapps.example.com'
export TUNAPI_PUBLIC_SCHEME='https'
export TUNAPI_PORT='8443'
export TUNAPI_ROUTES_FILE='./routes.json'
export TUNAPI_ALLOWED_TARGETS='127.0.0.1,localhost'
```

## 4) Run the Server

```bash
./tunapi
```

## 5) Register a Route (direct mode)

```bash
curl -sS -X POST 'http://127.0.0.1:8443/register' \
  -H 'X-Secret: your-secret-here' \
  -H 'Content-Type: application/json' \
  -d '{"subdomain":"demo","target":"127.0.0.1","port":8080}'
```

## 6) Test

```bash
curl -sS 'http://127.0.0.1:8443/' \
  -H 'Host: demo.tunapps.example.com'
```

## Next Steps

- [Install with systemd](./INSTALL.md) for production
- [Set up TunAgent](./TUNAGENT.md) for behind-NAT services
- [API reference](./API.md) for all endpoints
