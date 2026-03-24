# TunaAgent 🐟

> **Free your services. Swim upstream.**

Expose local services to the world — no inbound ports, no configuration, no cloud account needed. TunaAgent creates a secure reverse tunnel from your machine to your domain, powered by a WebSocket connection your machine initiates.

Open source. Always free.

---

## TL;DR

```bash
# Install the CLI
curl -fsSL https://get.tunaagent.dev/install.sh | bash

# One command to expose any local port
tuna connect --port 3000 --subdomain my-app

# Your app is live at:
# https://my-app.tuna.cloud
```

---

## What It Does

```
┌─────────────────────────────────────────────────────────────────┐
│                         YOUR MACHINE                              │
│                                                                   │
│   tuna connect --port 3000 --subdomain my-app                   │
│                    │                                              │
│              TunaAgent 🐟 ───────────────────────────►  wss://tuna.cloud
│                        (outbound :443)                           │
└─────────────────────────────────────────────────────────────────┘
                                                              │
                                                              ▼
                              ┌─────────────────────────────────────┐
                              │            TunaHub                  │
                              │     (your publicly hosted server)     │
                              │                                      │
                              │  nginx :443 → routes incoming        │
                              │  requests to the right TunaAgent     │
                              └─────────────────────────────────────┘
                                                              │
                                                              ▼
                              https://my-app.tuna.cloud
```

- **No inbound ports** — your machine only opens an outbound WebSocket
- **No cloud signup** — run your own TunaHub or use any public one
- **Works behind NAT** — home server, laptop, Raspberry Pi behind router
- **Zero config** — one command to expose any local port

---

## Install

```bash
# macOS / Linux (one-liner)
curl -fsSL https://get.tunaagent.dev/install.sh | bash

# Or download a release from GitHub
# https://github.com/andyeswong/tunapi/releases/latest

# Or build from source
git clone https://github.com/andyeswong/tunapi.git
cd tunapi
go build -o tuna-agent ./cmd/tunagent
go build -o tuna ./cmd/tunctl
go build -o tuna-server ./cmd/tunapi
sudo mv tuna-agent tuna tuna-server /usr/local/bin/
```

---

## Quick Start

### 1. Run a TunaHub (or use a public one)

```bash
# The server component that receives the tunnels
git clone https://github.com/andyeswong/tunaagent.git
cd tunaagent
go build -o tuna-server .
go build -o tuna .

# Run it
TUNA_SECRET=your-secret \
TUNA_BASE_DOMAIN=tuna.cloud \
./tuna-server
```

### 2. Register your first agent

```bash
# From another terminal, create an agent via the TunaHub API
./tuna agent create --name my-laptop
# Returns: {"id":"ag_xxx","name":"my-laptop","token":"tok_xxx"}
```

### 3. Connect your first service

```bash
# Login with the agent credentials
export TUNA_AGENT_ID=ag_xxx
export TUNA_AGENT_TOKEN=tok_xxx
export TUNA_SERVER=https://tuna.cloud

# Expose a local service
./tuna-agent

# Your app is now live at:
# https://my-laptop.tuna.cloud

# Check it's working
./tuna agent list
```

---

## Day-to-Day Shell Usage 🐚

```bash
# --- Setup aliases (add to ~/.bashrc or ~/.zshrc) ---
alias tuna='tuna-agent'
alias tunactl='tuna'

# tuna connect — expose a local port
tuna connect --port 5173 --subdomain frontend-dev

# tuna connect — expose a Node API
tuna connect --port 3000 --subdomain api-prod

# tuna connect — expose with a random subdomain (instant share)
tuna connect --port 8000 --random

# tuna connect — persistent tunnel as a systemd service
tuna connect --port 3000 --subdomain my-app --daemon

# tuna list — see all active tunnels
tuna list

# tuna agent — register a new agent
tuna agent create --name laptop
tuna agent list
tuna agent delete --name old-laptop

# --- Expose multiple services at once ---
tuna connect --port 3000  --subdomain api    &
tuna connect --port 5173  --subdomain web    &
tuna connect --port 5432  --subdomain pg     &
wait

# --- Share a local file server instantly ---
python3 -m http.server 8000 --directory ~/projects &
tuna connect --port 8000 --subdomain $(whoami)-files

# --- Expose a dev server, get the URL ---
URL=$(tuna connect --port 3000 --subdomain my-dev --json | jq -r .url)
echo "Share this: $URL"

# --- Expose SSH ( tunneling TCP, not HTTP) ---
# (TunaAgent forwards any TCP protocol, not just HTTP)
tuna connect --port 22 --subdomain my-pi --proto tcp

# --- Inspect and debug ---
tuna inspect api          # see connection stats
tuna logs                # tail agent logs
tuna status              # health check
```

---

## Systemd Service (Persistent Tunnel)

```bash
# Create the service file
sudo tee /etc/systemd/system/tuna-agent.service > /dev/null << 'EOF'
[Unit]
Description=TunaAgent — reverse tunnel
After=network.target

[Service]
Environment="TUNA_AGENT_ID=ag_xxx"
Environment="TUNA_AGENT_TOKEN=tok_xxx"
Environment="TUNA_SERVER=wss://tuna.cloud/agent/connect"
Environment="TUNA_SUBDOMAIN=my-app"
Environment="TUNA_PORT=3000"
ExecStart=/usr/local/bin/tuna-agent
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable tuna-agent
sudo systemctl start tuna-agent

# Monitor
sudo journalctl -u tuna-agent -f
```

---

## Components

| Binary | Role | Product Name |
|--------|------|-------------|
| `tuna-server` | Server (runs on your VPS) | **TunaHub** |
| `tuna-agent` | Client (runs on your machine) | **TunaAgent** 🐟 |
| `tuna` | Admin CLI (manage agents and routes) | **Tuna** |

---

## Running Your Own TunaHub

```bash
# Clone and build
git clone https://github.com/andyeswong/tunaagent.git
cd tunaagent

go build -o tuna-server .          # the server
go build -o tuna-agent ./cmd/tunagent  # the client
go build -o tuna ./cmd/tunctl         # the CLI

# Configure nginx (SSL termination)
sudo cp tuna.cloud.nginx.conf /etc/nginx/sites-available/tuna
sudo ln -s /etc/nginx/sites-available/tuna /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# Run TunaHub
TUNA_SECRET=change-me-in-production \
TUNA_BASE_DOMAIN=tuna.cloud \
./tuna-server
```

---

## Architecture

### Agent Mode (for NAT'ed clients)

Your machine initiates a WebSocket connection to TunaHub. TunaHub uses that connection to route incoming HTTP requests to your local service. **No inbound ports opened on your end.**

```
Internet ──► nginx :443 ──► TunaHub ──► active TunaAgent
                                        │
                                  ┌─────┴─────┐
                                  │            │
                                  ▼            ▼
                            TunaAgent ──► localhost:PORT
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

# Temporary share for debugging (random subdomain)
tuna connect --port 8000 --random

# Development webhook testing
tuna connect --port 3000 --subdomain webhook-debug

# Share a local ML model API
tuna connect --port 8000 --subdomain llama-api

# Access your Pi from anywhere
ssh pi@home
tuna connect --port 22 --subdomain my-pi

# Share a local tunnel for GitHub Actions / CI testing
tuna connect --port 3000 --subdomain ci-test --header "X-Debug: true"
```

---

## Environment Variables

```bash
# TunaAgent (client)
TUNA_AGENT_ID=ag_xxx          # Agent ID from tuna agent create
TUNA_AGENT_TOKEN=tok_xxx      # Agent token
TUNA_AGENT_NAME=my-laptop     # Human-readable name
TUNA_SERVER=wss://tuna.cloud/agent/connect  # TunaHub WebSocket endpoint
TUNA_SUBDOMAIN=my-app         # Default subdomain
TUNA_PORT=3000                # Default local port
TUNA_LOG_LEVEL=info           # debug | info | warn

# TunaHub (server)
TUNA_SECRET=change-me         # Admin secret
TUNA_BASE_DOMAIN=tuna.cloud    # Your base domain
TUNA_PORT=8443                # HTTP API port (default 8443)
```

---

## TunaCloud (Public Instance)

Free public TunaHub for anyone to use:

```bash
export TUNA_AGENT_ID=your-agent-id
export TUNA_AGENT_TOKEN=your-token
export TUNA_SERVER=wss://tuna.cloud/agent/connect

tuna-agent
```

> **Note:** Public instances have rate limits and are best-effort. For production, run your own TunaHub.

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

```bash
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
