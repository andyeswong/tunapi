# Troubleshooting

## Agent not connecting

1. Verify agent ID and token match the server's `agents.json`
2. Check the WebSocket URL is correct (`wss://.../agent/connect`)
3. Ensure outbound port 443 is open on the client
4. Check agent logs for auth errors

## Route returns 502

- For agent-mode routes: agent must be online
- For direct routes: verify target IP:port is reachable from the server
- Check the service on the client is running (`curl 127.0.0.1:<localPort>`)

## Route returns 404

- Subdomain not found in `routes.json`
- Check subdomain matches exactly (case-sensitive)

## Truncated responses / missing bytes

Fixed in latest version with `bufio.Writer` + `Flush()` for streaming.

Ensure both tunagent and tunapi-server are updated.

## tunapi-server won't start: "address already in use"

Another process is using port 8443:

```bash
ss -tlnp | grep 8443
kill <pid>
systemctl restart tunapi
```

## 401 unauthorized

- `/register`, `/list`, `/agent/list` require `X-Secret` header
- Verify `TUNAPI_SECRET` matches the one set in the systemd environment

## 403 Forbidden (target not allowed)

The target IP is not in `TUNAPI_ALLOWED_TARGETS`. Set it in environment:

```
TUNAPI_ALLOWED_TARGETS=127.0.0.1,localhost,192.168.1.0/24
```

## nginx http2 error: "unknown directive"

Some nginx builds don't support `http2` on `listen` directives. Remove `http2` from the listen line in the nginx site config.

## Certificate error (SSL)

- Verify the SSL cert exists at the path in the nginx config
- For Let's Encrypt: ensure port 80 is open for ACME challenges
- Check cert expiry: `openssl s_client -connect domain:443`

## Agent disconnects frequently

- Check network stability between client and server
- Ensure no firewall is idle-timing out the WebSocket connection
- Consider adding a ping keepalive interval in the agent code
