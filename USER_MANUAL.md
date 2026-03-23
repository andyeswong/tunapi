# Manual de Usuario

## Concepto
TunAPI enruta tráfico por subdominio:
- `app.tunapps.example.com` -> `target:port`

Cada ruta vive en un archivo JSON (`routes.json`) y se administra vía API.

## Flujo típico
1. Levantas tu app local (ej. `127.0.0.1:8080`)
2. Registras subdominio (`app`)
3. TunAPI comienza a proxear
4. Si ya no la quieres, borras la ruta

## API de uso diario

### Health
```bash
curl -sS http://127.0.0.1:8443/health
```

### Registrar
```bash
curl -sS -X POST 'http://127.0.0.1:8443/register' \
  -H 'X-Secret: TU_SECRET' \
  -H 'Content-Type: application/json' \
  -d '{"subdomain":"app","target":"127.0.0.1","port":8080}'
```

### Listar
```bash
curl -sS 'http://127.0.0.1:8443/list' -H 'X-Secret: TU_SECRET'
```

### Borrar
```bash
curl -sS -X DELETE 'http://127.0.0.1:8443/register?subdomain=app' \
  -H 'X-Secret: TU_SECRET'
```

## Reglas importantes
- `subdomain` solo minúsculas, números y guiones
- `target` debe estar en `TUNAPI_ALLOWED_TARGETS`
- `TUNAPI_SECRET` no puede ser `changeme`

## Buenas prácticas
- Usa un secret largo (32+ chars)
- Mantén `TUNAPI_ALLOWED_TARGETS` lo más estricta posible
- Pon Nginx/Caddy delante para TLS y headers
- Guarda backup de `routes.json`
