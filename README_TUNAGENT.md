# TunAgent — TunAPI Client Agent

TunAgent is the client component of TunAPI. It runs on machines behind NAT/firewall and connects via WebSocket to the TunAPI server, enabling reverse-tunnel access to local services.

## What It Does

```
[Your Device] ←──tunagent──→ [WebSocket] ←──tunapi-server──→ [Internet]
   (Apache)                     (port 80)                             
```

TunAgent connects **outbound** to the TunAPI server, so no inbound ports need to be opened on the client.

## Quick Start

### Prerequisites
- Go 1.21+
- Outbound access to port `443` (or the TunAPI server port)

### Build
```bash
cd ~/tunapi
go build -ldflags="-s -w" -o tunagent ./cmd/tunagent
```

### Run
```bash
# With environment variables
TUNAPI_AGENT_ID=ag_xxx \
TUNAPI_AGENT_TOKEN=your_token \
TUNAPI_AGENT_NAME=my-client \
TUNAPI_SERVER=wss://tunapps.example.com/agent/connect \
./tunagent
```

### With Embedded Defaults (pre-configured build)
```bash
./tunagent
# Uses built-in defaults for ID, token, server, and name
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|---|---|---|
| `TUNAPI_AGENT_ID` | `ag_mUXw_Fn3HPih0jyw` | Agent ID (from `agent/create`) |
| `TUNAPI_AGENT_TOKEN` | *(empty)* | Agent token (from `agent/create`) |
| `TUNAPI_AGENT_NAME` | `cliente-test` | Human-readable name |
| `TUNAPI_SERVER` | `wss://tunapps.andres-wong.com/agent/connect` | TunAPI server WebSocket URL |

### Setting Up a New Agent

1. **Create agent on server:**
```bash
curl -sS -X POST http://127.0.0.1:8443/agent/create \
  -H 'Content-Type: application/json' \
  -d '{"name":"mi-pc"}'
```

Response:
```json
{"id":"ag_abc123","name":"mi-pc","token":"raw_token_here"}
```

2. **Build tunagent with the credentials:**
```bash
# Edit cmd/tunagent/main.go defaults:
# var agentID    = "ag_abc123"
# var agentToken = "raw_token_here"
# var agentName  = "mi-pc"
# var serverURL  = "wss://tunapps.example.com/agent/connect"

go build -ldflags="-s -w" -o tunagent ./cmd/tunagent
```

3. **Publish the route:**
```bash
curl -sS -X POST http://127.0.0.1:8443/publish \
  -H 'Content-Type: application/json' \
  -H 'X-Secret: tunapi_secret_2026' \
  -d '{"subdomain":"mi-pc","agent":"mi-pc","localPort":80}'
```

4. **Run the agent** on the client machine:
```bash
./tunagent
```

5. **Access** via `https://mi-pc.tunapps.example.com`

## How It Works

### Connection Flow
1. TunAgent dials the TunAPI server WebSocket endpoint with auth headers
2. Server validates `X-Agent-ID` + `X-Agent-Token`
3. On success, agent is registered in the agent hub and kept alive with ping/pong
4. Server can now send `open_stream` messages to this agent

### Request Flow (Internet → Local Service)
```
Browser → nginx → tunapi-server → WebSocket → tunagent → Apache (localhost)
```

1. Request arrives at `https://subdomain.tunapps.example.com`
2. Nginx proxies to `tunapi-server` on `:8443`
3. Server looks up the route, finds it's an `agent` mode route
4. Server sends `open_stream` WebSocket message to the matching agent
5. TunAgent dials `127.0.0.1:<localPort>`, builds the HTTP request, sends it
6. TunAgent reads the HTTP response and streams chunks back via `stream_data`
7. Server assembles chunks and returns the HTTP response to the client

### Key Implementation Details

**HTTP Request Building** (`buildHTTPRequest`):
- Builds a complete HTTP/1.1 request: `GET /path HTTP/1.1\r\nHost: ...\r\n\r\n`
- Attaches all headers and body
- Writes directly to the TCP socket

**Stream Chunks:**
- Response body is encoded as base64 and sent as `stream_data` messages
- `more: true` = more chunks coming
- Final chunk triggers `stream_close`

## Running as a Service

### systemd (Linux)
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
sudo cp tunagent /usr/local/bin/
sudo cp tunagent.service /etc/systemd/system/
sudo systemctl enable tunagent
sudo systemctl start tunagent
```

### Docker
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
docker build -t tunagent .
docker run -d --restart unless-stopped \
  -e TUNAPI_AGENT_ID=ag_xxx \
  -e TUNAPI_AGENT_TOKEN=xxx \
  -e TUNAPI_AGENT_NAME=my-client \
  -e TUNAPI_SERVER=wss://tunapps.example.com/agent/connect \
  tunagent
```

## Security Notes

- **Token is sensitive** — treat it like a password
- Default/embedded credentials should be replaced for production
- Agent tokens are stored as SHA-256 hashes on the server side
- The WebSocket connection should use `wss://` (TLS) in production
- Consider using a firewall to restrict which local ports tunagent can expose

## Troubleshooting

### "connection failed: websocket dial: ..."
- Verify the server URL is correct and reachable
- Check that the agent ID and token are correct
- Ensure outbound port 443 (or server port) is open

### "agent connected successfully" but route returns 502
- Verify the `localPort` matches a service running on the client
- Check the service is listening on `127.0.0.1` (not just `localhost`)
- Ensure no firewall on the client is blocking local connections

### Request times out
- TunAgent may have crashed; check its logs
- The target service may be unresponsive; verify it's running

### Truncated responses / missing final bytes
- Ensure both tunagent and tunapi-server are updated to the latest version
- The fix uses `bufio.Writer` + `Flush()` for proper streaming
