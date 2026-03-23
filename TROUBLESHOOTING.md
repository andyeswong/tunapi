# Troubleshooting

## 401 unauthorized en /register o /list
- Revisa `X-Secret`
- Verifica que `TUNAPI_SECRET` en runtime sea el esperado

## 404 route not found
- La ruta no existe o expiró
- Valida con `GET /list` (con secret)

## 400 target not allowed
- Tu `target` no está en `TUNAPI_ALLOWED_TARGETS`
- Agrega el host/IP permitido y reinicia

## 404 en proxy por host
- El host no coincide con `*.TUNAPI_BASE_DOMAIN`
- Ejemplo: `demo.tunapps.example.com`

## No inicia por `changeme`
- Es esperado por seguridad
- Define `TUNAPI_SECRET` real

## Ver logs (systemd)
```bash
journalctl -u tunapi -f
```
