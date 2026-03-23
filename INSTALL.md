# Instalación

## Opción A: binario local

```bash
cd tunapi
go build -o tunapi .
sudo install -m 0755 tunapi /usr/local/bin/tunapi
```

## Opción B: systemd (recomendado)

### 1) Crear usuario de servicio
```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tunapi
```

### 2) Crear directorio de estado
```bash
sudo mkdir -p /etc/tunapi
sudo chown tunapi:tunapi /etc/tunapi
```

### 3) Archivo de entorno
`/etc/tunapi/tunapi.env`

```bash
TUNAPI_SECRET=pon-un-secreto-largo-y-unico
TUNAPI_BASE_DOMAIN=tunapps.example.com
TUNAPI_PUBLIC_SCHEME=https
TUNAPI_PORT=8443
TUNAPI_ROUTES_FILE=/etc/tunapi/routes.json
TUNAPI_ALLOWED_TARGETS=127.0.0.1,localhost
```

### 4) Unit file
`/etc/systemd/system/tunapi.service`

```ini
[Unit]
Description=TunAPI dynamic reverse-proxy control plane
After=network.target

[Service]
User=tunapi
Group=tunapi
EnvironmentFile=/etc/tunapi/tunapi.env
ExecStart=/usr/local/bin/tunapi
Restart=always
RestartSec=2
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/tunapi

[Install]
WantedBy=multi-user.target
```

### 5) Habilitar y arrancar
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now tunapi
sudo systemctl status tunapi
```

## Nginx de borde
Usa `tunapps.nginx.conf` como plantilla. Cambia dominio y upstream según tu entorno.
