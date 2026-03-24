# tunctl — Admin CLI

Command-line interface for managing TunAPI routes and agents.

## Build

```bash
go build -ldflags="-s -w" -o tunctl ./cmd/tunctl
```

## Environment

```bash
export TUNAPI_URL='http://127.0.0.1:8443'
export TUNAPI_SECRET='your-secret'
```

## Commands

### Health

```bash
tunctl health
```

### List routes

```bash
tunctl list
```

### Register route (direct mode)

```bash
tunctl register \
  --subdomain demo \
  --target 127.0.0.1 \
  --port 8080
```

### Delete route

```bash
tunctl delete --subdomain demo
```

### Agent: create

```bash
tunctl agent create --name my-client
# Output: {"id":"ag_xxx","name":"my-client","token":"raw_token"}
```

### Agent: list

```bash
tunctl agent list
```

### Agent: delete

```bash
tunctl agent delete --name my-client
```

### Publish: route via agent

```bash
tunctl publish \
  --subdomain my-client \
  --agent my-client \
  --local-port 80
```

### Unpublish

```bash
tunctl unpublish --subdomain my-client
```
