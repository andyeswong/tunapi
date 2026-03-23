# API Reference

Base URL: `http://127.0.0.1:8443`

## Auth
Para endpoints protegidos usa:
- Header: `X-Secret: <secret>`
- o query: `?secret=<secret>`

## GET /health
Respuesta:
```json
{"status":"ok"}
```

## POST /register (protegido)
Body:
```json
{
  "subdomain":"demo",
  "target":"127.0.0.1",
  "port":8080
}
```

## DELETE /register?subdomain=demo (protegido)
Elimina la ruta por subdominio.

## GET /list (protegido)
Lista rutas registradas.

## Proxy catch-all /
Con `Host: <subdomain>.<base_domain>`, enruta al `target:port` registrado.
