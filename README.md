# TunaAgent 🐟

> **Free your services. Swim upstream.**

Expose local services to the world — no inbound ports, no configuration, no cloud account needed. TunaAgent creates a secure reverse tunnel from your machine to your domain, powered by a WebSocket connection your machine initiates.

Open source. Always free.

---

## TL;DR

```bash
# Install
curl -fsSL https://get.tunaagent.dev | bash

# Connect your first service (one command)
tuna connect --port 3000 --subdomain my-app

# Done. Your app is live at:
# https://my-app.tunapps.example.com
```

---

## What It Does

```
┌─────────────────────────────────────────────────────────────────┐
│                         YOUR MACHINE                              │
│                                                                   │
│   tuna connect --port 3000 --subdomain my-app                   │
│                    │                                              │
│              TunaAgent                                           │
│                 🐟 ──────────────────────────────►  wss://tuna.cloud
│                        (outbound :443)                            │
└─────────────────────────────────────────────────────────────────┘
                                                              │
                                                              ▼
                              ┌─────────────────────────────────────┐
                              │            TunaHub                  │
                              │     (your publicly hosted server)     │
                              │                                      │
                              │  nginx :443 → routes incoming       │
                              │  requests to the right TunaAgent    │
                              └─────────────────────────────────────┘
                                                              │
                                                              ▼
                              https://my-app.tunapps.example.com
```

- **No inbound ports** — your machine only opens an outbound WebSocket
- **No cloud signup** — run your own TunaHub or use any public one
- **Works behind NAT** — home server, laptop, Raspberry Pi behind router
- **Zero config** — one command to expose any local port

---

## Quick Start

### 1. Install TunaAgent

```bash
# macOS / Linux
curl -fsSL https://get.tunaagent.dev | bash

# Or build from source
git clone https://github.com/andyeswong/tunaagent.git
cd tunaagent
go build -o tuna ./cmd/tunaagent
sudo mv tuna /usr/local/bin/
```

### 2. Run a TunaHub (or use a public one)

```bash
# The server component that receives the tunnels
git clone https://github.com/andyeswong/tunapi.git
cd tunapi
go build -o tuna-server ./cmd/tunapi

# Run it
TUNAPI_SECRET=your-secret \
TUNAPI_BASE_DOMAIN=tunapps.example.com \
./tuna-server
```

### 3. Connect your first service

```bash
# Register an agent (get your API key from the TunaHub admin)
tuna login https://tunapps.example.com --secret your-secret

# Expose a local service
tuna connect --port 3000 --subdomain my-app

# Your app is now live at:
# https://my-app.tunapps.example.com

# Check status
tuna status

# Disconnect
tuna disconnect
```

---

## Day-to-Day Shell Usage 🐚

```bash
# Expose a dev server
tuna connect --port 5173 --subdomain frontend-dev

# Expose a Node API
tuna connect --port 3000 --subdomain api-prod

# Expose multiple services at once
tuna connect --port 3000  --subdomain api    &
tuna connect --port 5173  --subdomain web     &
tuna connect --port 5432  --subdomain postgres &
wait

# See all active tunnels
tuna list

# Inspect a tunnel
tuna inspect api

# Share a local URL instantly
python3 -m http.server 8000 &
tuna connect --port 8000 --subdomain $(whoami)-share

# Persistent tunnel (add to rc.local or systemd)
tuna connect --port 3000 --subdomain my-app --daemon
```

---

## Components

| Component | Language | Description |
|-----------|----------|-------------|
| **TunaAgent** | Go | The client you run locally. Opens the WebSocket tunnel. |
| **TunaHub** | Go | The server. Receives tunnels, routes HTTP traffic. |
| **Tuna** | Go | The CLI. Login, connect, list, inspect, disconnect. |

### Running your own TunaHub

```bash
# Minimal production setup
git clone https://github.com/andyeswong/tunapi.git
cd tunapi

# Build all components
go build -o tuna-server .
go build -o tuna ./cmd/tunaagent
go build -o tunactl ./cmd/tunctl

# Run with environment variables
TUNAPI_SECRET=super-secret \
TUNAPI_BASE_DOMAIN=tunapps.example.com \
./tuna-server

# Register your first agent (from another terminal)
./tunctl agent create --name my-laptop

# Copy the agent credentials, then on your laptop:
tuna login https://tunapps.example.com --secret super-secret
tuna connect --port 3000 --subdomain my-app
```

---

## Architecture

### Agent Mode (for NAT'ed clients)

Your machine initiates a WebSocket connection to TunaHub. TunaHub uses that connection to route incoming HTTP requests to your local service. **No inbound ports opened on your end.**

```
Client (you) ──► TunaHub (your server)
   │                  │
   │  wss://tunapps   │  HTTPS from internet
   │  .example.com    │
   │                  ▼
   │            nginx :443
   │                  │
   │            routes to active TunaAgent
   │                  │
   ▼                  ▼
TunaAgent ──────► your localhost:PORT
(your machine)
```

### Direct Mode (for reachable IPs)

TunaHub connects directly to your server's IP. No agent needed, but your server must be reachable from TunaHub.

```
Internet ──► nginx ──► TunaHub ──► 192.168.1.100:80
                                    (your server, no agent)
```

---

## Use Cases

```bash
# Share a local demo with a client
tuna connect --port 3000 --subdomain demo-client-2024

# Expose a homelab service without port forwarding
tuna connect --port 8123 --subdomain homeassistant

# Temporary share for debugging (adds random subdomain)
tuna connect --port 8000 --random

# Development webhook testing (expose local dev server)
tuna connect --port 3000 --subdomain webhook-debug

# Share a local ML model API
tuna connect --port 8000 --subdomain llama-api

# Access your Pi from anywhere
ssh pi@home
tuna connect --port 22 --subdomain my-pi  # expose SSH
```

---

## TunaCloud (Public TunaHub)

Free public instance hosted by the community:

```bash
# Using the public TunaCloud
tuna login https://tunapps.andres-wong.com --secret tunapi_secret_2026
tuna connect --port 3000 --subdomain my-app
```

> **Note:** Public instances have rate limits and are best-effort. For production, run your own TunaHub.

---

## Project Structure

```
tunapi/
├── cmd/
│   ├── tunagent/          # TunaAgent (client)
│   └── tunctl/            # Tuna CLI (admin)
├── pkg/
│   ├── types/             # Shared types
│   └── ...
├── main.go                # TunaHub (server)
├── server.go              # HTTP API + WebSocket hub
├── ws.go                  # WebSocket handling
├── types.go               # Core types
├── README.md              # This file
└── docs/                  # Full documentation
```

---

## Full Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](./docs/GETTING_STARTED.md) | 5-minute setup |
| [Install](./docs/INSTALL.md) | Production deployment |
| [Server Operations](./docs/SERVER.md) | Running TunaHub |
| [TunaAgent](./docs/TUNAGENT.md) | Client setup |
| [CLI Reference](./docs/CLI.md) | `tuna` command reference |
| [API Reference](./docs/API.md) | TunaHub HTTP API |
| [Troubleshooting](./docs/TROUBLESHOOTING.md) | Common issues |

---

## Contributing

```
# Fork, clone, build
git clone https://github.com/your-handle/tunaagent.git
cd tunaagent
go build ./...

# Run tests
go test ./...

# Submit a PR
# All contributors become part of the Shoal 🐠
```

Open an Issue for bugs, feature requests, or just to say hi.

---

## License

MIT — free your ocean.

---

**"The fish that swims upstream most powerfully is the one that matters most."**
