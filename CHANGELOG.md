# Changelog

All notable changes to QoLauncher are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-07-01

First release: Go runtime + interactive deploy wrapper.

### Added

- **Launcher runtime** (`cmd/launcher`) as container PID 1
- **Configuration** via ENV and CLI flags ([docs/04-configuration.md](docs/04-configuration.md))
- **Process supervisor** with restart policies `never`, `on-failure`, `always`
- **Crash loop protection** via `APP_RESTART_BURST` + `APP_RESTART_WINDOW`
- **Graceful shutdown** with signal forwarding and `APP_SHUTDOWN_TIMEOUT`
- **Optional HTTP health check** probe for the application
- **Log capture** — stdout/stderr to daily files `YYYY-MM-DD.log`
- **Log retention** sweep on startup and every 24h
- **Log viewer** — Basic Auth, list/view/download logs on `LOG_PORT`
- **Structured launcher logs** to stderr with supervisor lifecycle events
- **Docker image** — multi-stage Alpine, non-root user
- **`launcher.sh`** — interactive deploy (Run/Stop/Restart/Status/Logs/Apps/Setup)
- First-run wizard: `.env` global + generated `docker-compose.yml`
- **Multi-app**: satu container per app; pilih `all` / satu / beberapa (`1,3`)
- **`apps/`** demo layout (`http-server`, `hello`) + Makefile targets
- **CI** — GitHub Actions `go test` + `go vet`
- **Documentation** — design docs in `docs/`

### Fixed

- Menu interaktif tetap terbuka saat tidak ada container / aksi dibatalkan
- Banner menampilkan **Running** (live Docker), bukan label Active dari state file
- **`make build-examples`** memakai `CGO_ENABLED=0` — binary demo kompatibel dengan image Alpine (hindari restart loop `exec format error`)
- Demo **`http-server`** membaca `APP_PORT` dari env (selaras dengan healthcheck & compose)
- Log **`application start failed`** saat binary tidak bisa dijalankan (mis. dynamic glibc di Alpine)

### Changed

- Example apps in `apps/` (not `examples/`)
- `docker-compose.yml` generated locally, gitignored
- Dokumentasi: build static untuk Docker, pemisahan log launcher vs output app, port demo di `apps/*/.env`

### Security

- Log viewer protected with HTTP Basic Auth (constant-time password compare)
- `LOG_PASSWORD` redacted in `--config` output
- `LAUNCHER_*` env vars not passed to child process

### Verified (MVP Definition of Done)

| Check | Status |
|-------|--------|
| Deploy with `on-failure` | Supervisor integration tests |
| Auto restart on failure | Covered by supervisor integration tests |
| Logs in daily file + viewer | Implemented |
| Graceful `docker stop` | `stop_grace_period: 35s` + signal forwarding |
| Crash loop guard | Integration test (`TestSupervisorCrashLoopGuard`) |
| Viewer Basic Auth | Integration tests in `internal/viewer` |
| Exit code propagation (`never`) | Supervisor tests |
| `--config` / `--version` | CLI tests |
| Unit tests pass | `make test` |

### Known limitations

- Single application per container (multi-app = N containers via `launcher.sh`)
- Log viewer per app on separate port — no unified folder picker (planned V3)
- No log tail/search/pagination (planned V2)
- No Prometheus metrics (planned V2)
- Log viewer HTTP only — use reverse proxy for TLS in production

---

## Release checklist

To publish a release:

```bash
git tag -a v0.1.0 -m "QoLauncher MVP v0.1.0"
git push origin v0.1.0
```

Build and push image (example):

```bash
docker build -t ghcr.io/raqolbi/qolauncher:0.1.0 -t ghcr.io/raqolbi/qolauncher:latest .
docker push ghcr.io/raqolbi/qolauncher:0.1.0
docker push ghcr.io/raqolbi/qolauncher:latest
```

[0.1.0]: https://github.com/raqolbi/QoLauncher/releases/tag/v0.1.0
