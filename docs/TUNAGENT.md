# TunAgent — Client Agent

TunAgent runs on machines behind NAT/firewall. It connects via WebSocket to the TunAPI server and forwards HTTP requests to local services.

## Quick Start

### Build

```bash
go build -ldflags="-s -w" -o tunagent ./cmd/tunagent
```

### Run with environment variables

```bash
TUNAPI_AGENT_ID=ag_xxx \
TUNAPI_AGENT_TOKEN=raw_token \
TUNAPI_AGENT_NAME=my-client \
TUNAPI_SERVER=wss://tunapps.example.com/agent/connect \
./tunagent
```

### Run with embedded defaults

Edit `cmd/tunagent/main.go` and set defaults:

```go
var agentID    = "ag_xxx"
var agentToken = "raw_token"
var agentName  = "my-client"
var serverURL  = "wss://tunapps.example.com/agent/connect"
```

Then build and run without env vars:

```bash
./tunagent
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `TUNAPI_AGENT_ID` | — | Agent ID from `agent/create` |
| `TUNAPI_AGENT_TOKEN` | — | Agent token from `agent/create` |
| `TUNAPI_AGENT_NAME` | — | Human-readable name |
| `TUNAPI_SERVER` | — | WebSocket URL of TunAPI server |

## Setup a New Agent

1. Create agent on server:
```bash
curl -sS -X POST http://127.0.0.1:8443/agent/create \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-client"}'
```

2. Embed the returned `{id, name, token}` in `cmd/tunagent/main.go` defaults.

3. Build and deploy to client:
```bash
go build -ldflags="-s -w" -o tunagent ./cmd/tunagent
scp tunagent user@client:/home/user/tunagent
ssh user@client "./tunagent"
```

4. Publish the route:
```bash
curl -sS -X POST http://127.0.0.1:8443/publish \
  -H 'X-Secret: your-secret' \
  -H 'Content-Type: application/json' \
  -d '{"subdomain":"my-client","agent":"my-client","localPort":80}'
```

5. Access via `https://my-client.tunapps.example.com`

## How It Works

1. TunAgent dials `wss://server/agent/connect` with auth headers
2. Server validates and registers the agent in the hub
3. Server sends `open_stream` message when a request comes in
4. TunAgent dials `127.0.0.1:<localPort>`, sends HTTP request
5. TunAgent streams response chunks back via `stream_data`
6. Final chunk triggers `stream_close`

## systemd Service (recommended)

`/etc/systemd/system/tunagent.service`:

```ini
[Unit]
Description=TunAPI Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/tunagent
Restart=always
RestartSec=5
EnvironmentFile=/etc/tunagent/env

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now tunagent
```

## Docker

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o tunagent ./cmd/tunagent

FROM alpine:latest
COPY --from=builder /app/tunagent /usr/local/bin/tunagent
ENTRYPOINT ["tunagent"]
```

```bash
docker run -d --restart unless-stopped \
  -e TUNAPI_AGENT_ID=ag_xxx \
  -e TUNAPI_AGENT_TOKEN=xxx \
  -e TUNAPI_AGENT_NAME=my-client \
  -e TUNAPI_SERVER=wss://tunapps.example.com/agent/connect \
  tunagent
```
