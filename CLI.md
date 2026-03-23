# tunctl (CLI Admin Interface)

`tunctl` es una interfaz de administración para TunAPI.

## Compilar
```bash
cd tunapi
go build -o tunctl ./cmd/tunctl
```

## Variables opcionales
- `TUNAPI_URL` (default `http://127.0.0.1:8443`)
- `TUNAPI_SECRET`

## Comandos

### Health
```bash
tunctl health
```

### Listar rutas
```bash
tunctl list --secret "$TUNAPI_SECRET"
```

### Registrar ruta
```bash
tunctl register \
  --subdomain demo \
  --target 127.0.0.1 \
  --port 8080 \
  --secret "$TUNAPI_SECRET"
```

### Eliminar ruta
```bash
tunctl delete --subdomain demo --secret "$TUNAPI_SECRET"
```

## Ejemplo con env
```bash
export TUNAPI_URL='http://127.0.0.1:8443'
export TUNAPI_SECRET='super-secret'

tunctl list
tunctl register --subdomain app --target 127.0.0.1 --port 3000
tunctl delete --subdomain app
```
