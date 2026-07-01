# QoLauncher — Overview

## Apa itu QoLauncher?

**QoLauncher** adalah runtime Docker universal untuk menjalankan aplikasi Go yang sudah di-build (binary). QoLauncher berperan sebagai **PID 1** di dalam container, menjalankan binary Go sebagai child process, dan menyediakan fitur operasional yang umum dibutuhkan di production tanpa mengubah source code aplikasi.

QoLauncher **bukan** framework Go. Aplikasi tetap ditulis seperti biasa — bahkan se-sederhana `http.ListenAndServe` atau `fmt.Println` — dan cukup di-copy ke container sebagai executable.

## Masalah yang Ingin Diselesaikan

| Masalah | Solusi QoLauncher |
|---------|-------------------|
| Binary Go tidak menulis log ke file; stdout/stderr hilang saat container restart | Capture stdout/stderr ke log harian dengan rotasi otomatis |
| Tidak ada cara mudah melihat log historis di container minimal | Web log viewer dengan Basic Auth |
| Graceful shutdown sering tidak ditangani di container | Signal forwarding + timeout ke child process |
| Aplikasi crash perlu restart tanpa orchestrator | Process Supervisor dengan restart policy |
| Tidak ada crash loop protection di container sederhana | Burst limit + sliding window guard |
| Setiap tim menulis Dockerfile/script sendiri untuk hal yang sama | Satu image runtime, cukup mount binary |
| Aplikasi sederhana tidak perlu di-refactor untuk production | Zero code change pada aplikasi Go |

## Target User

- **Developer Go** yang ingin deploy binary ke Docker tanpa menambah library logging atau supervisor ke aplikasi.
- **DevOps / SRE** yang butuh container image standar dengan logging, log viewer, dan graceful shutdown out-of-the-box.
- **Tim kecil / startup** yang ingin path deploy cepat dari `go build` ke production.
- **Maintainer open source** yang ingin menyediakan image Docker siap pakai untuk release binary.

## Non-Goal

QoLauncher **tidak** akan:

- Menjadi framework atau library yang di-import ke aplikasi Go.
- Menggantikan orchestrator (Kubernetes, Docker Swarm) untuk scaling, service discovery, atau load balancing.
- Mengubah behavior aplikasi (middleware HTTP, auto-instrumentasi, dll.) tanpa konfigurasi eksplisit.
- Menyediakan database, message queue, atau layanan sidecar lain.
- Menjadi process manager multi-app **di dalam satu container** (multi-app = beberapa container via `launcher.sh`; lihat [08-docker.md](./08-docker.md)).
- Menggantikan reverse proxy production (Traefik, Nginx); QoLauncher hanya expose log viewer, HTTPS diurus di luar container.
- Melakukan build Go di dalam container runtime (binary harus sudah di-build sebelumnya).

## Contoh Penggunaan

### Aplikasi HTTP sederhana

Contoh aplikasi Go:

```go
package main

import "net/http"

func main() {
    http.ListenAndServe(":8080", nil)
}
```

Build dan deploy:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o apps/my-api/server .
./launcher.sh   # wizard + menu interaktif (disarankan)
```

Binary untuk container Alpine **wajib static** (`CGO_ENABLED=0`). Lihat [08-docker.md](./08-docker.md).

Atau `docker run` langsung:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .
docker run -d \
  -v $(pwd)/server:/app/server:ro \
  -v $(pwd)/logs/my-api:/var/log/qolauncher \
  -e APP_BINARY=/app/server \
  -e APP_PORT=8080 \
  -e LOG_USERNAME=admin \
  -e LOG_PASSWORD=secret \
  -p 8080:8080 \
  -p 8081:8081 \
  qolauncher:latest
```

### Aplikasi CLI / worker

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello World")
}
```

Sama flow-nya; `APP_PORT=0` jika aplikasi tidak listen HTTP (hanya log viewer). Output `fmt.Println` masuk file log harian (`[stdout] Hello World`), bukan `docker compose logs`.

### Multi-app dengan `launcher.sh`

Taruh setiap binary di `apps/<app-id>/` + optional `apps/<app-id>/.env` (port unik). Satu container QoLauncher per app.

```
apps/
  http-server/server   + .env   → app :9998, viewer :9999, logs/logs/http-server/
  hello/hello          + .env   → viewer :9997, logs/logs/hello/
```

```bash
make build-examples
./launcher.sh          # Run → all | 1 | 1,2
```

### docker-compose (manual / generated)

`launcher.sh` men-generate `docker-compose.yml` otomatis. Contoh service tunggal manual:

```yaml
services:
  qolauncher-http-server:
    image: qolauncher:latest
    volumes:
      - ./apps/http-server/server:/app/server:ro
      - ./logs/http-server:/var/log/qolauncher
    env_file:
      - .env
      - apps/http-server/.env
    environment:
      APP_BINARY: /app/server
    ports:
      - "8080:8080"
      - "8081:8081"
```

## Prinsip Desain

1. **Convention over configuration** — ENV dengan default sensible; binary path wajib, sisanya optional.
2. **PID 1 responsibility** — launcher menangani signal, process supervision, dan graceful shutdown.
3. **Separation of concerns** — aplikasi hanya business logic; launcher hanya operasional.
4. **Security by default** — log viewer tidak accessible tanpa auth.

## Dokumen Terkait

| Dokumen | Isi |
|---------|-----|
| [02-architecture.md](./02-architecture.md) | Arsitektur dan lifecycle |
| [03-runtime.md](./03-runtime.md) | Detail eksekusi runtime |
| [04-configuration.md](./04-configuration.md) | Referensi ENV lengkap |
| [08-docker.md](./08-docker.md) | Dockerfile, `launcher.sh`, deployment |
| [11-process-supervisor.md](./11-process-supervisor.md) | Process Supervisor, restart policy, health check |
| [12-development-tasks.md](./12-development-tasks.md) | Development task tracker |
