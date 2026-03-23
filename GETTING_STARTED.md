# Getting Started (5 minutos)

Este quickstart está orientado a mantener TunAPI simple.

## 1) Requisitos
- Linux/macOS
- Go 1.23+
- Un dominio wildcard apuntando a tu servidor (ej. `*.tunapps.example.com`)
- Nginx (opcional pero recomendado para borde HTTP/HTTPS)

## 2) Compilar
```bash
cd tunapi
go build -o tunapi .
```

## 3) Variables mínimas
```bash
export TUNAPI_SECRET='cambia-esto-por-un-secreto-largo'
export TUNAPI_BASE_DOMAIN='tunapps.example.com'
export TUNAPI_PUBLIC_SCHEME='https'
export TUNAPI_PORT='8443'
export TUNAPI_ROUTES_FILE='./routes.json'
export TUNAPI_ALLOWED_TARGETS='127.0.0.1,localhost'
```

## 4) Ejecutar
```bash
./tunapi
```

## 5) Registrar una ruta
```bash
curl -sS -X POST 'http://127.0.0.1:8443/register' \
  -H 'X-Secret: cambia-esto-por-un-secreto-largo' \
  -H 'Content-Type: application/json' \
  -d '{"subdomain":"demo","target":"127.0.0.1","port":8080}'
```

## 6) Probar proxy por host
```bash
curl -sS 'http://127.0.0.1:8443/' -H 'Host: demo.tunapps.example.com'
```

## Endpoints API
- `GET /health`
- `POST /register` (auth)
- `DELETE /register?subdomain=...` (auth)
- `GET /list` (auth)

Auth: header `X-Secret` o query `?secret=`.

## (Opcional) CLI admin
```bash
go build -o tunctl ./cmd/tunctl

./tunctl health
./tunctl list --secret 'cambia-esto-por-un-secreto-largo'
./tunctl register --subdomain demo --target 127.0.0.1 --port 8080 --secret 'cambia-esto-por-un-secreto-largo'
```
