# API Reference

Base URL: `http://127.0.0.1:8443`

## Auth

Protected endpoints require:
- Header: `X-Secret: <secret>`
- Or query: `?secret=<secret>`

## Endpoints

### `GET /health`

Health check. No auth required.

```bash
curl http://127.0.0.1:8443/health
```

**Response:** `{"status":"ok"}`

---

### `POST /agent/create`

Register a new agent. Returns `{id, name, token}`. Token is shown only once.

```bash
curl -sS -X POST http://127.0.0.1:8443/agent/create \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-client"}'
```

**Response:**
```json
{"id":"ag_xxx","name":"my-client","token":"raw_token_here"}
```

---

### `GET /agent/list`

List all registered agents. Auth required.

```bash
curl http://127.0.0.1:8443/agent/list \
  -H 'X-Secret: your-secret'
```

**Response:**
```json
{
  "agents": [
    {"id":"ag_xxx","name":"my-client","online":true,"lastSeen":"2026-03-23T10:00:00Z","streams":0}
  ]
}
```

---

### `DELETE /agent/{id}`

Unregister an agent. Auth required.

---

### `POST /publish`

Publish a subdomain via an online agent.

```bash
curl -sS -X POST http://127.0.0.1:8443/publish \
  -H 'Content-Type: application/json' \
  -H 'X-Secret: your-secret' \
  -d '{"subdomain":"my-client","agent":"my-client","localPort":80}'
```

**Response:** `{"url":"https://my-client.tunapps.example.com"}`

---

### `DELETE /route/{subdomain}`

Remove an agent-mode route.

---

### `POST /register`

Register a direct-mode route (server proxies directly to target).

```bash
curl -sS -X POST http://127.0.0.1:8443/register \
  -H 'X-Secret: your-secret' \
  -H 'Content-Type: application/json' \
  -d '{"subdomain":"demo","target":"192.168.1.10","port":8080}'
```

**Response:** `{"url":"https://demo.tunapps.example.com","subdomain":"demo","target":"192.168.1.10:8080"}`

---

### `GET /list`

List all routes. Auth required.

---

### `DELETE /register?subdomain=demo`

Delete a direct-mode route. Auth required.

---

### `WS /agent/connect`

WebSocket endpoint for tunagent to connect.

Headers required: `X-Agent-ID`, `X-Agent-Token`, `X-Agent-Name`
