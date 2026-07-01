# Konfigurasi

QoLauncher dikonfigurasi via **environment variables** (production) dan **CLI flags** (development). CLI menimpa ENV; ENV menimpa default.

Semua fitur MVP — termasuk Process Supervisor — dikonfigurasi melalui ENV. Detail supervisor: [11-process-supervisor.md](./11-process-supervisor.md).

## Ringkasan ENV

### Aplikasi

| Variable | Required | Default | Since |
|----------|----------|---------|-------|
| `APP_BINARY` | **Yes** | — | MVP |
| `APP_ARGS` | No | `""` | MVP |
| `APP_PORT` | No | `8080` | MVP |
| `APP_WORKDIR` | No | dir of `APP_BINARY` | MVP |

### Process Supervisor

| Variable | Required | Default | Since |
|----------|----------|---------|-------|
| `APP_RESTART_POLICY` | No | `never` | MVP |
| `APP_RESTART_DELAY` | No | `3s` | MVP |
| `APP_MAX_RESTART` | No | `0` | MVP |
| `APP_RESTART_WINDOW` | No | `60s` | MVP |
| `APP_RESTART_BURST` | No | `10` | MVP |
| `APP_SHUTDOWN_TIMEOUT` | No | `30s` | MVP |

### Logging

| Variable | Required | Default | Since |
|----------|----------|---------|-------|
| `LOG_DIR` | No | `/var/log/qolauncher` | MVP |
| `LOG_RETENTION_DAYS` | No | `14` | MVP |
| `LOG_LEVEL` | No | `info` | MVP |
| `TZ` | No | `UTC` | MVP |

### Log Viewer

| Variable | Required | Default | Since |
|----------|----------|---------|-------|
| `LOG_PORT` | No | `8081` | MVP |
| `LOG_USERNAME` | **Yes*** | — | MVP |
| `LOG_PASSWORD` | **Yes*** | — | MVP |
| `VIEWER_ENABLED` | No | `true` | MVP |

### Health Check (Opsional)

| Variable | Required | Default | Since |
|----------|----------|---------|-------|
| `HEALTHCHECK_ENABLED` | No | `false` | MVP |
| `HEALTHCHECK_TYPE` | No | `http` | MVP |
| `HEALTHCHECK_URL` | **Yes**** | — | MVP |
| `HEALTHCHECK_INTERVAL` | No | `30s` | MVP |
| `HEALTHCHECK_TIMEOUT` | No | `5s` | MVP |
| `HEALTHCHECK_FAILURES` | No | `3` | MVP |

\*Required when `VIEWER_ENABLED=true` (default).

\**Required when `HEALTHCHECK_ENABLED=true`.

---

## Aplikasi

### `APP_BINARY`

Path absolut atau relatif ke executable Go di dalam container.

```bash
APP_BINARY=/app/server
```

- Harus menunjuk file regular, bukan directory.
- Typical pattern: volume mount read-only ke path ini.
- Tidak ada auto-discovery; path eksplisit.

### `APP_ARGS`

Argumen command-line untuk binary, sebagai string tunggal.

```bash
APP_ARGS="--port 8080 --verbose"
```

**Parsing rules (MVP):**

- Split dengan `strings.Fields` (whitespace, no shell quoting sophistication).
- Untuk argumen dengan spasi, gunakan config file (V3+).
- Empty string → no args.

### `APP_PORT`

Port HTTP (atau primary listen port) aplikasi. Digunakan untuk:

- Dokumentasi / default health URL hint
- Docker `EXPOSE` guidance
- **Tidak** digunakan launcher untuk bind; aplikasi bind sendiri.

```bash
APP_PORT=8080
```

Validasi: integer 1–65535. Jika aplikasi non-HTTP, set ke `0`.

### `APP_WORKDIR`

Working directory saat menjalankan child.

```bash
APP_WORKDIR=/app
```

Default: parent directory dari `APP_BINARY`.

---

## Process Supervisor

### `APP_RESTART_POLICY`

Kebijakan restart setelah child exit.

```bash
APP_RESTART_POLICY=always
```

| Value | Perilaku |
|-------|----------|
| `never` | Tidak restart; launcher exit dengan exit code child |
| `on-failure` | Restart jika exit code ≠ 0 |
| `always` | Selalu restart kecuali intentional shutdown |

Detail: [11-process-supervisor.md](./11-process-supervisor.md).

### `APP_RESTART_DELAY`

Jeda sebelum start ulang child setelah exit.

```bash
APP_RESTART_DELAY=3s
```

Format: Go `time.ParseDuration`.

### `APP_MAX_RESTART`

Batas total restart sejak launcher start.

```bash
APP_MAX_RESTART=0
```

- `0` = unlimited (crash loop guard tetap berlaku).
- `N > 0` = stop setelah N restart, launcher exit `1`.

### `APP_RESTART_WINDOW`

Sliding window untuk deteksi crash loop.

```bash
APP_RESTART_WINDOW=60s
```

### `APP_RESTART_BURST`

Max restart dalam `APP_RESTART_WINDOW` sebelum launcher fatal exit.

```bash
APP_RESTART_BURST=10
```

Contoh: 10 restart dalam 60 detik → launcher exit `1`.

### `APP_SHUTDOWN_TIMEOUT`

Durasi tunggu child exit setelah signal sebelum `SIGKILL`.

```bash
APP_SHUTDOWN_TIMEOUT=30s
```

Format: Go `time.ParseDuration`. Harus < Docker `stop_grace_period`.

---

## Logging

### `LOG_DIR`

Direktori penyimpanan log harian aplikasi (stdout/stderr capture).

```bash
LOG_DIR=/var/log/qolauncher
```

- Dibuat otomatis jika belum ada (`0755`).
- Disarankan volume mount agar log persist across container recreate.

### `LOG_RETENTION_DAYS`

Jumlah hari file log dipertahankan sebelum dihapus.

```bash
LOG_RETENTION_DAYS=14
```

- `0` = tidak pernah hapus otomatis.
- Berdasarkan tanggal di **nama file**, bukan mtime.

### `LOG_LEVEL`

Level log **internal launcher / supervisor** (bukan aplikasi).

```bash
LOG_LEVEL=info
```

| Value | Output |
|-------|--------|
| `debug` | Verbose init, capture stats, supervisor state |
| `info` | Startup, shutdown, restart events |
| `warn` | Crash, forced shutdown |
| `error` | Crash loop, max restart |

Output ke stderr container.

### `TZ`

Timezone IANA untuk rotasi log harian.

```bash
TZ=Asia/Jakarta
```

Mempengaruhi penamaan `YYYY-MM-DD.log` saat rollover midnight.

---

## Log Viewer

### `LOG_PORT`

Port HTTP log viewer.

```bash
LOG_PORT=8081
```

Bind `0.0.0.0` (all interfaces) di dalam container. Lihat [07-security.md](./07-security.md).

### `LOG_USERNAME`

Username Basic Auth viewer.

```bash
LOG_USERNAME=admin
```

### `LOG_PASSWORD`

Password Basic Auth viewer.

```bash
LOG_PASSWORD=change-me-in-production
```

- **Wajib** diganti di production.
- Tidak loggable oleh launcher.
- Tidak diteruskan ke child process.

### `VIEWER_ENABLED`

Enable/disable HTTP log viewer.

```bash
VIEWER_ENABLED=true
```

`false` → viewer tidak start; auth ENV tidak required.

---

## Health Check (Opsional MVP)

### `HEALTHCHECK_ENABLED`

```bash
HEALTHCHECK_ENABLED=true
```

Default `false`. Aktifkan hanya untuk aplikasi HTTP dengan endpoint health.

### `HEALTHCHECK_TYPE`

```bash
HEALTHCHECK_TYPE=http
```

MVP: hanya `http`. Future: `tcp`.

### `HEALTHCHECK_URL`

```bash
HEALTHCHECK_URL=http://127.0.0.1:8080/health
```

Full URL untuk HTTP GET probe. Wajib jika enabled.

### `HEALTHCHECK_INTERVAL`

```bash
HEALTHCHECK_INTERVAL=30s
```

### `HEALTHCHECK_TIMEOUT`

```bash
HEALTHCHECK_TIMEOUT=5s
```

Per-request timeout.

### `HEALTHCHECK_FAILURES`

```bash
HEALTHCHECK_FAILURES=3
```

Consecutive failures sebelum stop child dan apply restart policy.

---

## Reserved Environment

Variabel dengan prefiks `LAUNCHER_` reserved untuk internal launcher dan **tidak** diteruskan ke child process.

---

## Future (Reserved)

Documented for consistency; **not implemented MVP**.

### `LOG_GZIP_AFTER_DAYS` (V2)

Compress older logs.

### `METRICS_PORT` (V2)

Prometheus scrape port.

### `VIEWER_ALLOW_CIDRS` (V2)

IP allowlist for log viewer.

---

## CLI Mapping

| ENV | CLI Flag |
|-----|----------|
| `APP_BINARY` | `--binary` |
| `APP_ARGS` | `--args` |
| `APP_PORT` | `--app-port` |
| `APP_WORKDIR` | `--workdir` |
| `APP_RESTART_POLICY` | `--restart-policy` |
| `APP_RESTART_DELAY` | `--restart-delay` |
| `APP_MAX_RESTART` | `--max-restart` |
| `APP_RESTART_WINDOW` | `--restart-window` |
| `APP_RESTART_BURST` | `--restart-burst` |
| `APP_SHUTDOWN_TIMEOUT` | `--shutdown-timeout` |
| `LOG_DIR` | `--log-dir` |
| `LOG_PORT` | `--viewer-port` |
| `LOG_RETENTION_DAYS` | `--log-retention-days` |
| `LOG_USERNAME` | `--log-username` |
| `LOG_PASSWORD` | `--log-password` |
| `LOG_LEVEL` | `--log-level` |
| `TZ` | `--tz` |
| `VIEWER_ENABLED` | `--viewer-enabled` |
| `HEALTHCHECK_ENABLED` | `--healthcheck-enabled` |
| `HEALTHCHECK_TYPE` | `--healthcheck-type` |
| `HEALTHCHECK_URL` | `--healthcheck-url` |
| `HEALTHCHECK_INTERVAL` | `--healthcheck-interval` |
| `HEALTHCHECK_TIMEOUT` | `--healthcheck-timeout` |
| `HEALTHCHECK_FAILURES` | `--healthcheck-failures` |

Detail perintah: [09-cli.md](./09-cli.md).

---

## Konfigurasi per app (`launcher.sh`)

Saat deploy via [08-docker.md](./08-docker.md) / `launcher.sh`:

| File | Scope |
|------|-------|
| `.env` (root) | Default shared: `LOG_USERNAME`, `LOG_PASSWORD`, policy default, dll. |
| `apps/<app-id>/.env` | Override per app: **`APP_PORT`**, **`LOG_PORT`**, restart policy, healthcheck |

Docker Compose menggunakan `env_file: [.env, apps/<id>/.env]` — nilai di file app menimpa global.

**Multi-app:** setiap app **wajib** punya `LOG_PORT` unik; `APP_PORT` unik jika expose HTTP (`0` jika worker tanpa HTTP).

Contoh `apps/http-server/.env`:

```bash
APP_PORT=8080
LOG_PORT=8081
HEALTHCHECK_URL=http://127.0.0.1:8080/health
```

Contoh `apps/hello/.env`:

```bash
APP_PORT=0
LOG_PORT=8082
HEALTHCHECK_ENABLED=false
```

---

## Contoh `.env`

```bash
APP_BINARY=/app/server
APP_ARGS=
APP_PORT=8080
APP_WORKDIR=/app

APP_RESTART_POLICY=on-failure
APP_RESTART_DELAY=3s
APP_MAX_RESTART=0
APP_RESTART_WINDOW=60s
APP_RESTART_BURST=10
APP_SHUTDOWN_TIMEOUT=30s

LOG_DIR=/var/log/qolauncher
LOG_PORT=8081
LOG_RETENTION_DAYS=30
LOG_USERNAME=admin
LOG_PASSWORD=super-secret
LOG_LEVEL=info

TZ=UTC
VIEWER_ENABLED=true

HEALTHCHECK_ENABLED=true
HEALTHCHECK_URL=http://127.0.0.1:8080/health
HEALTHCHECK_INTERVAL=30s
HEALTHCHECK_TIMEOUT=5s
HEALTHCHECK_FAILURES=3
```

---

## Validasi Error Messages

| Error | Cause |
|-------|-------|
| `APP_BINARY is required` | Missing binary path |
| `APP_BINARY not found` | Stat failed |
| `invalid APP_RESTART_POLICY` | Not never/on-failure/always |
| `LOG_DIR is not writable` | Permission |
| `LOG_USERNAME and LOG_PASSWORD required when viewer enabled` | Auth missing |
| `HEALTHCHECK_URL required when healthcheck enabled` | Missing URL |
| `invalid APP_PORT` | Non-numeric |
| `APP_RESTART_BURST must be >= 1` | Invalid burst |

Semua fatal pre-start.
