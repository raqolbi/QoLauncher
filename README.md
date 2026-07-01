# QoLauncher

Universal Docker runtime for Go binaries — run any compiled Go app without changing source code.

QoLauncher acts as **PID 1** inside the container: it starts your binary, captures stdout/stderr to daily log files, supervises restarts, forwards signals for graceful shutdown, and serves a password-protected log viewer.

**Repository:** [github.com/raqolbi/QoLauncher](https://github.com/raqolbi/QoLauncher)

## Features (v0.1.0)

- Zero code change — mount your `go build` output and run
- Process supervisor with restart policies: `never`, `on-failure`, `always`
- Crash loop protection (configurable burst + window)
- Daily log files with retention cleanup
- Web log viewer (Basic Auth): list, view, download
- Optional HTTP health check probe
- Signal forwarding: `SIGTERM`, `SIGINT`, `SIGQUIT`
- Configuration via environment variables or CLI flags
- **`launcher.sh`** — interactive deploy wrapper (multi-app, auto compose, wizard setup)

## Quick Start

### Cara termudah — `launcher.sh` (interaktif)

```bash
# Contoh bawaan repo (boleh hapus isi apps/ dan ganti app sendiri)
make build-examples

# Atau app kamu sendiri (wajib static binary untuk image Alpine):
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o apps/my-api/server .

# (Optional) env khusus app — wajib port unik untuk multi-app
cp .env.example apps/my-api/.env

# Jalankan menu interaktif
./launcher.sh
```

Menu: **Run** · **Stop** · **Restart** · **Status** · **Logs** · **Apps** · **Setup**

| Menu | Pilihan app |
|------|-------------|
| **Run** | Semua app di `apps/` — ketik `all`, satu nomor, atau `1,3` |
| **Stop / Restart / Logs** | Hanya app yang **sedang jalan** — pilihan sama (`all` / satu / beberapa) |

- **Multi-app:** satu container Docker per app; `docker-compose.yml` di-generate otomatis
- Pertama kali: wizard buat `.env` global + `docker-compose.yml`
- Auto-detect binary di `apps/` (abaikan `main.go` / source)
- **Run** → container background; menu bisa ditutup
- Banner **`Running:`** = container live (bukan cache deploy lama)

Detail: [docs/08-docker.md](docs/08-docker.md) · [apps/README.md](apps/README.md)

### Docker Compose via Makefile

```bash
make compose-up    # build examples + image + jalankan semua app
make compose-down
```

| App | URL (port dari `apps/<id>/.env` bawaan) |
|-----|----------------------------------------|
| http-server | http://localhost:9998 |
| http-server viewer | http://localhost:9999/logs |
| hello viewer | http://localhost:9997/logs |

> **`docker compose logs`** hanya menampilkan log **launcher** (supervisor). Output app (`fmt.Println`, log HTTP, dll.) ada di **`logs/<app-id>/YYYY-MM-DD.log`** atau log viewer.

## Run your own binary (manual)

Tanpa `launcher.sh`, build binary untuk Linux lalu mount ke container:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o apps/my-api/server .
./launcher.sh   # disarankan — auto-generate compose

# atau edit docker-compose.yml manual / docker run:
make docker-build
```

Lihat [docs/08-docker.md](docs/08-docker.md) untuk `docker run` dan compose manual.

### Local dev (CLI, tanpa Docker)

```bash
make build
./bin/launcher \
  --binary ./bin/myapp \
  --log-dir ./logs \
  --log-username admin \
  --log-password secret \
  --restart-policy on-failure
```

## Docker (manual)

```bash
make docker-build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

docker run -d --name api \
  -v $(pwd)/server:/app/server:ro \
  -v $(pwd)/logs:/var/log/qolauncher \
  -e APP_BINARY=/app/server \
  -e APP_RESTART_POLICY=on-failure \
  -e LOG_USERNAME=admin \
  -e LOG_PASSWORD=secret \
  -p 8080:8080 -p 8081:8081 \
  qolauncher:latest
```

## CLI

```bash
launcher --help
launcher --version
launcher --config --binary ./bin/app --log-password secret
```

See [docs/09-cli.md](docs/09-cli.md) for all flags.

## Configuration

All settings are environment variables. See [docs/04-configuration.md](docs/04-configuration.md) or `.env.example`.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `APP_BINARY` | Yes | — | Path to Go executable |
| `APP_RESTART_POLICY` | No | `never` | `never`, `on-failure`, `always` |
| `LOG_DIR` | No | `/var/log/qolauncher` | Daily log directory |
| `LOG_PORT` | No | `8081` | Log viewer port |
| `LOG_USERNAME` | Yes* | — | Viewer Basic Auth user |
| `LOG_PASSWORD` | Yes* | — | Viewer Basic Auth password |

\*Not required when `VIEWER_ENABLED=false`.

## Examples

Demo apps live in `apps/` (same layout as production). You can delete them and use your own binaries.

| App | Build | Ports (demo `.env`) |
|-----|-------|---------------------|
| `http-server` | `make build-examples` | app :9998, viewer :9999 |
| `hello` | (same) | viewer :9997 (app sekali jalan lalu exit) |

Run all examples:

```bash
make compose-up
# or interactively: ./launcher.sh → Run → all apps
```

## Development

```bash
make build      # bin/launcher
make test       # unit tests
make lint       # go vet / golangci-lint
```

Project layout:

```
launcher.sh            Interactive deploy wrapper (entry point)
apps/                  User binaries + bundled examples
cmd/launcher/          Go launcher entrypoint (PID 1 in container)
internal/config/       ENV + CLI configuration
internal/supervisor/   Process supervisor
internal/logwriter/    Daily log files
internal/capture/      stdout/stderr capture
internal/viewer/       HTTP log viewer
internal/logger/       Launcher structured logs
docs/                  Design documentation
```

## Documentation

| Doc | Topic |
|-----|-------|
| [docs/01-overview.md](docs/01-overview.md) | Goals & use cases |
| [docs/04-configuration.md](docs/04-configuration.md) | ENV reference |
| [docs/06-log-viewer.md](docs/06-log-viewer.md) | Log viewer (per-app) |
| [docs/08-docker.md](docs/08-docker.md) | Docker + `launcher.sh` |
| [docs/11-process-supervisor.md](docs/11-process-supervisor.md) | Supervisor design |
| [docs/12-development-tasks.md](docs/12-development-tasks.md) | Dev task tracker |

## License

MIT — see [LICENSE](LICENSE).
