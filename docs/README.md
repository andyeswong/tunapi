# TunaAgent Documentation 🐟

> **Free your services. Swim upstream.**

## Quick Navigation

| Guide | Description |
|-------|-------------|
| [Getting Started](./GETTING_STARTED.md) | 5-minute setup guide |
| [Install](./INSTALL.md) | Production installation (systemd, nginx) |
| [Server Operations](./SERVER.md) | Running TunaHub in production |
| [TunaAgent](./TUNAGENT.md) | Client agent setup & deployment |
| [CLI Reference](./CLI.md) | `tuna` command reference |
| [API Reference](./API.md) | TunaHub HTTP API |
| [Troubleshooting](./TROUBLESHOOTING.md) | Common issues & fixes |

## Architecture Overview

```
Internet → nginx (SSL) → TunaHub :8443 → target
                                       ↓
                             WebSocket tunnel
                                       ↓
                             TunaAgent (client) → local service
```

- **TunaHub**: Runs on a public VPS. Routes HTTP requests via WebSocket to TunaAgent.
- **TunaAgent**: Runs on a client behind NAT. Opens WebSocket to server, receives requests, forwards to local service.
- **Tuna**: Admin CLI for managing agents and routes.

## Quick Start

```bash
# Install
curl -fsSL https://get.tunaagent.dev/install.sh | bash

# Build from source
git clone https://github.com/andyeswong/tunaagent.git
cd tunaagent
go build -o tuna-server .
go build -o tuna-agent ./cmd/tunagent
go build -o tuna ./cmd/tunctl

# Run TunaHub
TUNA_SECRET=my-secret \
TUNA_BASE_DOMAIN=tuna.cloud \
./tuna-server

# Register an agent (another terminal)
./tuna agent create --name my-laptop

# Connect from your machine
export TUNA_AGENT_ID=ag_xxx
export TUNA_AGENT_TOKEN=tok_xxx
export TUNA_SERVER=wss://tuna.cloud/agent/connect
./tuna-agent
```

## Project Structure

```
tunaagent/
├── cmd/
│   ├── tunagent/          # TunaAgent (client binary)
│   └── tunctl/            # Tuna CLI (admin binary)
├── pkg/
│   └── types/             # Shared types
├── main.go                # TunaHub (server binary)
├── server.go              # HTTP API + WebSocket hub
├── ws.go                  # WebSocket handling
├── types.go               # Core types
├── README.md              # Main README
├── install.sh             # One-line install script
└── docs/                  # Full documentation
```

## Key Files

| File | Description |
|------|-------------|
| `main.go` | TunaHub server entry point |
| `server.go` | Agent API (create/list/delete/publish) |
| `ws.go` | WebSocket hub + HTTP routing via agent |
| `types.go` | AgentStore, AgentHub, persistence |
| `cmd/tunagent/main.go` | TunaAgent client |
| `cmd/tunctl/main.go` | Tuna admin CLI |
