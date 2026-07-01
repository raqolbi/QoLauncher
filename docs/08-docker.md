# Docker Deployment

Desain image Docker, compose, volume, dan networking untuk QoLauncher.

## Deploy dengan `launcher.sh` (disarankan)

`launcher.sh` di root repo adalah **entry point interaktif** untuk deploy tanpa edit compose manual.

### Alur

1. Copy binary Go (Linux) ke `apps/<app-id>/` (+ optional `apps/<app-id>/.env`)
2. Jalankan `./launcher.sh`
3. Pertama kali: wizard buat `.env` global + `docker-compose.yml`
4. Menu **Run** → pilih app (`all` / satu nomor / `1,3`) → container jalan di background

### Struktur `apps/`

```
apps/
  my-api/
    server          ← binary (GOOS=linux)
    .env            ← override port & policy per app
  worker/
    worker
    .env
```

Repo bawaan berisi demo `apps/http-server/` dan `apps/hello/` (source + `.env`). Isi folder **boleh dihapus** dan diganti app sendiri.

### Multi-app

- **Satu container QoLauncher per app** (bukan multi-process dalam satu container)
- `launcher.sh` men-generate `docker-compose.yml` dengan satu service per app
- Setiap app wajib **port unik** (`APP_PORT`, `LOG_PORT`) di `apps/<id>/.env`
- Log host: `logs/<app-id>/` per app
- Viewer: port terpisah per app (lihat [06-log-viewer.md](./06-log-viewer.md))

### Menu interaktif

| Menu | Daftar app | Pilihan |
|------|------------|---------|
| Run | Semua di `apps/` | `all`, `2`, `1,3` |
| Stop / Restart / Logs | Hanya yang **sedang running** | `all`, `2`, `1,3` |
| Status | — | Semua service di compose |
| Setup | — | Wizard ulang `.env` global |

Banner menu menampilkan **`Running:`** (status live dari Docker), bukan daftar deploy lama. Jika tidak ada container jalan: `Running: (tidak ada)`. Error atau batal di menu → kembali ke menu (tidak exit script).

### File yang di-ignore (git)

Tidak di-push: `/.env`, `docker-compose.yml`, `.launcher/`, `logs/`, binary di `apps/`. Lihat `.gitignore`.

### Non-interaktif (Makefile / CI)

```bash
make build-examples    # → apps/http-server/server, apps/hello/hello
make compose-up        # cp .env.example + ./launcher.sh --run-all
make compose-down      # ./launcher.sh --stop
```

| Flag | Fungsi |
|------|--------|
| `./launcher.sh --run-all` | Deploy semua app tanpa prompt |
| `./launcher.sh --stop` | Hentikan semua container |
| `./launcher.sh --help` | Bantuan |

State deploy disimpan di `.launcher/state`. `docker-compose.yml` di-overwrite saat Run/Setup.

---

Single **runtime image** containing only QoLauncher binary (static Go build). Application binary **not** baked in — mounted at runtime.

```
┌─────────────────────────────────┐
│  qolauncher:latest              │
│  ─────────────────────────────  │
│  /usr/local/bin/launcher        │  ← entrypoint
│  (optional) ca-certificates     │
│  USER qolauncher (non-root)     │
└─────────────────────────────────┘
         ▲
         │ mount
    /app/server  (host binary)
```

## Dockerfile (Design)

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /launcher ./cmd/launcher

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 qolauncher
COPY --from=builder /launcher /usr/local/bin/launcher
USER qolauncher
ENTRYPOINT ["/usr/local/bin/launcher"]
# No CMD — config via ENV
```

Notes:

- `tzdata` supports `TZ` env.
- Distroless alternative acceptable for minimal attack surface (V2 doc variant).

## Expose Ports

| Port | ENV | Purpose |
|------|-----|---------|
| `8080` | `APP_PORT` | Application (informational EXPOSE) |
| `8081` | `LOG_PORT` | Log viewer |

```dockerfile
EXPOSE 8080 8081
```

`EXPOSE` dokumentatif; `-p` required at run time.

## Volumes

### Binary mount

```bash
-v /host/path/server:/app/server:ro
```

Set `APP_BINARY=/app/server`.

### Log persistence

```bash
-v /host/path/logs:/var/log/qolauncher
```

Set `LOG_DIR=/var/log/qolauncher` (default matches).

Ensure UID `1000` can write mounted dir on host.

### Optional workdir / config

```bash
-v /host/config:/app/config:ro
-e APP_WORKDIR=/app
-e APP_ARGS="--config /app/config/app.yaml"
```

## docker-compose.yml

File di root **di-generate oleh `launcher.sh`**. Contoh service tunggal (manual):

```yaml
services:
  qolauncher-myapp:
    image: qolauncher:latest
    build: .
    restart: unless-stopped
    env_file:
      - .env
      - apps/myapp/.env
    volumes:
      - ./apps/myapp/server:/app/server:ro
      - ./logs/myapp:/var/log/qolauncher
    environment:
      APP_BINARY: /app/server
      APP_PORT: "8080"
      LOG_PORT: "8081"
    ports:
      - "8080:8080"
      - "8081:8081"
    stop_grace_period: 35s
```

### `stop_grace_period`

Harus ≥ `APP_SHUTDOWN_TIMEOUT` + buffer agar Docker tidak SIGKILL launcher sebelum child selesai.

## Run Examples

### Minimal

```bash
docker run -d --name api \
  -v $(pwd)/server:/app/server:ro \
  -e APP_BINARY=/app/server \
  -e LOG_USERNAME=admin \
  -e LOG_PASSWORD=secret \
  -p 8080:8080 -p 8081:8081 \
  qolauncher:latest
```

### Logs only on volume (no published viewer)

```bash
docker run -d \
  -v $(pwd)/server:/app/server:ro \
  -v $(pwd)/logs:/var/log/qolauncher \
  -e APP_BINARY=/app/server \
  -e VIEWER_ENABLED=false \
  -p 8080:8080 \
  qolauncher:latest
```

When viewer is disabled, `LOG_USERNAME` and `LOG_PASSWORD` are not required.

### Build app + run

```bash
GOOS=linux GOARCH=amd64 go build -o apps/my-api/server .
./launcher.sh                    # interaktif
# atau:
make compose-up                  # demo apps + --run-all
```

## HEALTHCHECK (Future V2)

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8081/health || exit 1
```

MVP optional; checks launcher viewer not app health.

## Multi-arch

Build with `docker buildx`:

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t qolauncher:latest .
```

## CI/CD Pattern

1. Build Go app binary for `linux/$ARCH`.
2. Push binary artifact.
3. Deploy container with QoLauncher image + binary mount.

No need to rebuild QoLauncher image per app release.

## Multi-app networking

```
                    ┌─────────────────────────────────────┐
  User ────────────►│  http-server :8080  viewer :8081    │
                    │  hello         viewer :8082         │
                    └─────────────────────────────────────┘
                              logs/http-server/
                              logs/hello/
```

Production: jangan publish port viewer ke publik; gunakan VPN / internal network.

## Referensi

- ENV: [04-configuration.md](./04-configuration.md)
- Log viewer per app: [06-log-viewer.md](./06-log-viewer.md)
- Security: [07-security.md](./07-security.md)
- Overview: [01-overview.md](./01-overview.md)
- Folder apps: [../apps/README.md](../apps/README.md)
