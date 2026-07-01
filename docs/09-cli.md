# CLI Reference

QoLauncher menyediakan CLI untuk menjalankan launcher di luar Docker (local dev) dan override konfigurasi ENV.

**Status:** Spesifikasi desain — belum diimplementasi.

## Invocation

```bash
launcher [flags]
```

Binary name: `launcher` (Docker: `/usr/local/bin/launcher`).

## Global Flags

| Flag | ENV Equivalent | Default | Description |
|------|----------------|---------|-------------|
| `--help` | — | — | Show usage |
| `--version` | — | — | Print version and exit |
| `--config` | — | — | Print resolved config and exit |
| `--binary` | `APP_BINARY` | — | Path to Go application binary |
| `--args` | `APP_ARGS` | `""` | Arguments for application |
| `--app-port` | `APP_PORT` | `8080` | Application port metadata |
| `--workdir` | `APP_WORKDIR` | parent of binary | Working directory for app |
| `--restart-policy` | `APP_RESTART_POLICY` | `never` | `never`, `on-failure`, `always` |
| `--restart-delay` | `APP_RESTART_DELAY` | `3s` | Delay before restart |
| `--max-restart` | `APP_MAX_RESTART` | `0` | Max lifetime restarts; 0=unlimited |
| `--restart-window` | `APP_RESTART_WINDOW` | `60s` | Crash loop sliding window |
| `--restart-burst` | `APP_RESTART_BURST` | `10` | Max restarts in window |
| `--shutdown-timeout` | `APP_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |
| `--log-dir` | `LOG_DIR` | `/var/log/qolauncher` | Log output directory |
| `--viewer-port` | `LOG_PORT` | `8081` | Log viewer HTTP port |
| `--log-retention-days` | `LOG_RETENTION_DAYS` | `14` | Log retention in days |
| `--log-username` | `LOG_USERNAME` | — | Viewer Basic Auth user |
| `--log-password` | `LOG_PASSWORD` | — | Viewer Basic Auth password |
| `--log-level` | `LOG_LEVEL` | `info` | Launcher log level |
| `--tz` | `TZ` | `UTC` | Timezone for log rotation |
| `--viewer-enabled` | `VIEWER_ENABLED` | `true` | Enable log viewer |
| `--healthcheck-enabled` | `HEALTHCHECK_ENABLED` | `false` | Enable health probe |
| `--healthcheck-type` | `HEALTHCHECK_TYPE` | `http` | Probe type |
| `--healthcheck-url` | `HEALTHCHECK_URL` | — | Health check URL |
| `--healthcheck-interval` | `HEALTHCHECK_INTERVAL` | `30s` | Probe interval |
| `--healthcheck-timeout` | `HEALTHCHECK_TIMEOUT` | `5s` | Probe timeout |
| `--healthcheck-failures` | `HEALTHCHECK_FAILURES` | `3` | Failures before restart |

Priority: **CLI > ENV > default**.

## Commands / Modes

MVP: single run mode only (no subcommands).

### Default run

```bash
launcher --binary ./server --log-username admin --log-password dev
```

Starts supervisor + viewer; blocks until launcher exit.

### `--version`

```bash
$ launcher --version
QoLauncher v0.1.0 (commit abc1234, built 2026-07-01T00:00:00Z)
```

Exit code: `0`.

### `--help`

```bash
$ launcher --help
QoLauncher - Universal Docker runtime for Go binaries

Usage:
  launcher [flags]

Flags:
  --binary string           Path to application binary (required)
  ...
```

Exit code: `0`.

### `--config`

Print effective configuration (secrets redacted):

```bash
$ launcher --config --binary ./server --log-password secret
APP_BINARY=./server
APP_ARGS=
APP_PORT=8080
APP_WORKDIR=.
APP_RESTART_POLICY=never
APP_RESTART_DELAY=3s
APP_MAX_RESTART=0
APP_RESTART_WINDOW=60s
APP_RESTART_BURST=10
APP_SHUTDOWN_TIMEOUT=30s
LOG_DIR=/var/log/qolauncher
LOG_PORT=8081
LOG_RETENTION_DAYS=14
LOG_USERNAME=
LOG_PASSWORD=***REDACTED***
LOG_LEVEL=info
TZ=UTC
VIEWER_ENABLED=true
HEALTHCHECK_ENABLED=false
```

Does **not** start app or viewer. Exit code: `0` if valid, `1` if validation fails.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success / help / version / intentional shutdown |
| 1 | Config error, crash loop, max restart exceeded |
| N | Child exit code N (policy `never`) |
| 128+S | Child killed by signal S |

See [03-runtime.md](./03-runtime.md) and [11-process-supervisor.md](./11-process-supervisor.md).

## Local Development Examples

### Run with restart on failure

```bash
go build -o ./bin/server .
launcher \
  --binary ./bin/server \
  --restart-policy on-failure \
  --app-port 8080 \
  --viewer-port 8081 \
  --log-dir ./logs \
  --log-username admin \
  --log-password admin
```

### Hello World worker (no restart)

```bash
GOOS=linux GOARCH=amd64 go build -o ./bin/hello ./apps/hello
launcher --binary ./bin/hello --log-dir ./logs/hello --log-username a --log-password b
```

### ENV-only (mirrors Docker)

```bash
export APP_BINARY=./bin/server
export APP_RESTART_POLICY=on-failure
export LOG_USERNAME=admin
export LOG_PASSWORD=admin
launcher
```

## Flag Parsing Library

Implementation recommendation: `flag` stdlib or `spf13/pflag` for GNU-style long flags.

Validation runs after parse, before any side effects.

## Future CLI (V2+)

| Command | Purpose |
|---------|---------|
| `launcher validate` | Config check only |
| `launcher rotate` | Manual retention sweep |
| `launcher version --json` | Machine-readable version |

Not in MVP scope.

## Referensi

- ENV detail: [04-configuration.md](./04-configuration.md)
- Runtime: [03-runtime.md](./03-runtime.md)
- Supervisor: [11-process-supervisor.md](./11-process-supervisor.md)
