# TunAPI

Dynamic subdomain reverse-proxy with WebSocket tunnel support.

Expose services from machines behind NAT using a reverse tunnel — no inbound ports needed on the client.

## Documentation

**[→ Start here: docs/README.md](./docs/README.md)**

Quick links:
- [Getting Started](./docs/GETTING_STARTED.md) — 5-minute setup
- [Install](./docs/INSTALL.md) — Production deployment
- [Server Operations](./docs/SERVER.md) — Managing the server
- [TunAgent](./docs/TUNAGENT.md) — Client agent setup
- [API Reference](./docs/API.md)
- [CLI (tunctl)](./docs/CLI.md)
- [Troubleshooting](./docs/TROUBLESHOOTING.md)

## Quick Start

```bash
git clone https://github.com/andyeswong/tunapi.git
cd tunapi

# Build
go build -o tunapi .
go build -o tunagent ./cmd/tunagent
go build -o tunctl ./cmd/tunctl

# Run server
TUNAPI_SECRET=my-secret TUNAPI_BASE_DOMAIN=tunapps.example.com ./tunapi
```

See [Getting Started](./docs/GETTING_STARTED.md) for full guide.

## Architecture

```
Internet → nginx (SSL) → tunapi-server :8443
                                  ↓
                          WebSocket tunnel
                                  ↓
                          tunagent (client) → local service
```

- **tunapi-server**: Routes HTTP requests through WebSocket tunnels
- **tunagent**: Client agent, runs behind NAT, connects outbound to server
- **tunctl**: Admin CLI for managing routes and agents

## Features

- Dynamic route registration via API
- Direct mode: server proxies directly to target IP
- Agent mode: reverse tunnel via WebSocket (for NAT'ed clients)
- Route persistence in JSON
- Agent persistence with token auth
- CLI and HTTP API

## Production

See [Install guide](./docs/INSTALL.md) for systemd + nginx setup.
