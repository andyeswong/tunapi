# TunAPI Documentation

## Quick Navigation

| Guide | Description |
|---|---|
| [Getting Started](./GETTING_STARTED.md) | 5-minute setup guide |
| [Install](./INSTALL.md) | Production installation (systemd, nginx) |
| [Server Operations](./SERVER.md) | Managing tunapi-server in production |
| [API Reference](./API.md) | HTTP API endpoints |
| [CLI (tunctl)](./CLI.md) | Admin CLI reference |
| [TunAgent](./TUNAGENT.md) | Client agent setup & deployment |
| [Troubleshooting](./TROUBLESHOOTING.md) | Common issues & fixes |

## Architecture Overview

```
Internet → nginx (SSL) → tunapi-server :8443 → target
                                        ↓
                              WebSocket tunnel
                                        ↓
                              tunagent (client) → local service
```

- **tunapi-server**: Runs on a public VPS. Routes HTTP requests via WebSocket to tunagent.
- **tunagent**: Runs on a client behind NAT. Opens WebSocket to server, receives requests, forwards to local service.
- **tunctl**: Admin CLI for managing routes and agents.

## Key Files

| File | Description |
|---|---|
| `main.go` | tunapi-server entry point |
| `server.go` | Agent API (create/list/delete/publish) |
| `ws.go` | WebSocket hub + HTTP routing via agent |
| `types.go` | AgentStore, AgentHub, persistence |
| `pkg/types/route.go` | Route + RouteMode (direct/agent) |
| `pkg/types/ws.go` | WS message types |
| `cmd/tunagent/main.go` | Client agent |
| `cmd/tunctl/main.go` | Admin CLI |
