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
┌─────────────────────────────────────────────────────────────────────────┐
│                              INTERNET                                     │
│                   https://myapp.tunapps.example.com                      │
└──────────────────────────────────┬────────────────────────────────────────┘
                                   │ TLS (port 443)
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  tunapi-server (VPS / public IP)            tunapi-server: 100.65.86.28 │
│  ┌─────────────┐     ┌──────────────┐     ┌─────────────────────────┐   │
│  │    nginx    │ ──▶ │   :8443      │     │   AgentHub (in-memory)  │   │
│  │  (SSL term) │     │  HTTP server │     │   ┌─────────────────┐   │   │
│  │  port 80/443│     │  + WS hub    │     │   │  cliente-test   │   │   │
│  └─────────────┘     └──────────────┘     │   │  ag_mUXw_...    │   │   │
│                                           │   │  WS connected ✓ │   │   │
│  Routes stored in:                        │   └─────────────────┘   │   │
│  /etc/tunapi/routes.json                  │   Agents persisted in:   │   │
│  /etc/tunapi/agents.json                  │   /etc/tunapi/agents.json│   │
└───────────────────────────────────────────┼───────────────────────────┘   │
                                            │                               │
                         ┌──────────────────┴──────┐                       │
                         │   WebSocket tunnel       │                       │
                         │   wss://tunapps.example  │                       │
                         │   .com/agent/connect     │                       │
                         └──────────────────┬──────┘                       │
                                            │ outbound :443 (client INITIATES)
                         ┌──────────────────┴──────┐                       │
                         │                           │                       │
                    ┌────┴──────────────────┐  ┌───┴────────────┐          │
                    │  tunapi-server         │  │ tunapi-server  │          │
                    │  (another agent)       │  │ (another agent) │          │
                    └────────────────────────┘  └────────────────┘          │
                                            │                                │
                         ┌──────────────────┴──────┐                        │
                         │   Client network         │                        │
                         │   (behind NAT/firewall)  │                        │
                         │   No inbound ports open  │                        │
                         │                           │                        │
                    ┌────┴──────────────────┐  ┌───┴────────────┐          │
                    │  tunagent             │  │ tunagent       │          │
                    │  (cliente-test)        │  │ (another-agent) │          │
                    │  192.168.35.89         │  │ 10.0.0.50       │          │
                    └────┬──────────────────┘  └──┬────────────┘          │
                         │                           │                       │
                    ┌────┴──────────────────┐  ┌───┴────────────┐          │
                    │  Apache / Nginx        │  │ Service         │          │
                    │  localhost:80          │  │ localhost:8080   │          │
                    │  (web server)          │  │ (API, app, etc.) │          │
                    └────────────────────────┘  └─────────────────┘          │
                                                                         │
                         MODE A: Agent mode (NAT)  ◄─────────────────────────┘
                         MODE B: Direct mode (no agent)


┌─────────────────────────────────────────────────────────────────────────┐
│  DIRECT MODE (no agent needed)                                          │
│                                                                         │
│  Internet → nginx → tunapi-server :8443 ──▶ 192.168.35.129:80           │
│                                       (direct TCP)     (no agent)       │
└─────────────────────────────────────────────────────────────────────────┘
```

## Traffic Flow

### Mode A — Agent mode (reverse tunnel, for NAT'ed clients)

```
1.  User browses to: https://myapp.tunapps.example.com/

2.  DNS: myapp.tunapps.example.com → A → public IP of tunapi-server

3.  nginx receives on :443 (SSL termination)
    - Looks up myapp subdomain in routes.json
    - Routes to 127.0.0.1:8443 (tunapi-server)

4.  tunapi-server receives HTTP request
    - Matches Host header → route for "myapp"
    - Route mode = ModeAgent → looks up agent "myapp" in hub
    - Finds tunagent connected via WebSocket
    - Sends {type: "open_stream"} message to tunagent via WS

5.  tunagent (on client, behind NAT) receives open_stream
    - Opens TCP to localhost:80 (its local web server)
    - Sends raw HTTP request: "GET / HTTP/1.1\r\nHost: myapp.tunapps.example.com\r\n..."

6.  Local web server (Apache/Nginx) responds

7.  tunagent reads response bytes, sends stream_data chunks back via WS
    - WS message: {type: "stream_data", streamId: "st_xxx", data: [bytes], eof: false}

8.  tunapi-server receives chunks, forwards to nginx → client

9.  tunagent sends final chunk: {type: "stream_data", streamId: "st_xxx", eof: true}

10. tunapi-server sends final response to nginx → user sees the web page
```

### Mode B — Direct mode (server proxies straight to target IP)

```
1.  User browses to: https://demo.tunapps.example.com/

2.  nginx → tunapi-server :8443 (same as above steps 2-3)

3.  tunapi-server matches route for "demo"
    - Route mode = ModeDirect
    - Opens TCP directly to 192.168.35.129:80
    - Pipes request/response without any agent

4.  Response returns straight through tunapi-server → nginx → user
```

## Key Difference

| | Agent mode | Direct mode |
|---|---|---|
| Agent needed | Yes (tunagent on client) | No |
| Client network | Behind NAT / firewall | Accessible from server IP |
| Connection direction | Client → Server (outbound) | Server → Client (inbound) |
| Inbound ports on client | **0** (fully private) | 1 (target port open) |
| Use case | Laptops, home servers, NAT networks | VPS, co-lo, reachable IPs |

## Components

| Component | Where it runs | What it does |
|---|---|---|
| **nginx** | tunapi-server (public VPS) | SSL termination, routes :443 → :8443 |
| **tunapi-server** | tunapi-server (public VPS) | HTTP routing, WS hub, agent registry |
| **tunctl** | Anywhere (admin laptop, server) | CLI to manage routes and agents |
| **tunagent** | Client (behind NAT) | Opens WS tunnel, forwards to local service |

## Production

See [Install guide](./docs/INSTALL.md) for systemd + nginx setup.
