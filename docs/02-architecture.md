# Arsitektur QoLauncher

## Diagram Komponen

```
┌─────────────────────────────────────────────────────────────┐
│                        Docker Container                      │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              QoLauncher (PID 1)                         │ │
│  │                                                         │ │
│  │  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │ │
│  │  │  Config     │  │   Process    │  │  Log Writer    │ │ │
│  │  │  Loader     │  │  Supervisor  │  │  (daily files) │ │ │
│  │  └──────┬──────┘  └──────┬───────┘  └───────┬────────┘ │ │
│  │         │                │                   │          │ │
│  │         │         ┌──────▼───────┐           │          │ │
│  │         │         │ Pipe Capture │───────────┘          │ │
│  │         │         │ stdout/stderr│                      │ │
│  │         │         └──────┬───────┘                      │ │
│  │         │                │                              │ │
│  │  ┌──────▼──────┐  ┌──────▼───────┐  ┌────────────────┐ │ │
│  │  │ Log Viewer  │  │ Go Binary    │  │ Log Rotator    │ │ │
│  │  │ HTTP server │  │ (child proc) │  │ (retention)    │ │ │
│  │  └─────────────┘  └──────────────┘  └────────────────┘ │ │
│  │                        ▲                                │ │
│  │              Health Check Probe (optional)               │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  Volumes: binary mount, LOG_DIR mount (optional)             │
│  Ports: APP_PORT (app), LOG_PORT (viewer)                    │
└─────────────────────────────────────────────────────────────┘
```

## Alur Data

```
Docker
  │
  ▼
Launcher (PID 1)
  ├── membaca konfigurasi (ENV / CLI)
  ├── validasi (binary exists, executable)
  ├── Process Supervisor
  │     ├── menjalankan binary Go ──► stdout/stderr ──► pipe capture
  │     ├── monitor exit / restart policy
  │     ├── crash loop guard
  │     └── health check probe (optional)
  │                                    │
  │                                    ▼
  ├── log harian ◄────────────── Log Writer ──► YYYY-MM-DD.log
  ├── log rotator (startup + periodic)
  ├── log viewer HTTP ◄──────── reads LOG_DIR
  ├── graceful shutdown (SIGTERM/SIGINT/SIGQUIT)
  └── signal forwarding (to child)
```

## Modul Internal

| Modul | Tanggung jawab |
|---------|----------------|
| `config` | Parse ENV + CLI flags, validate |
| `supervisor` | Start, monitor, restart, crash guard, health probe |
| `process` | exec.Cmd wrapper, signal, wait exit |
| `capture` | Multiplex stdout/stderr dengan prefix |
| `logwriter` | Daily file naming, concurrent write |
| `rotator` | Delete files older than retention |
| `viewer` | HTTP server, Basic Auth, file listing |
| `shutdown` | Orchestrate graceful stop |

Modul-modul ini berjalan dalam satu proses Go; log viewer dan health probe sebagai goroutine terpisah.

Detail supervisor: [11-process-supervisor.md](./11-process-supervisor.md).

## Lifecycle Launcher

### Fase 1 — Init

1. Container start, `launcher` binary executed as entrypoint.
2. Parse configuration (ENV + CLI).
3. Initialize launcher logger (`LOG_LEVEL`); log `launcher started`.
4. Ensure `LOG_DIR` exists.
5. Run log rotation sweep (delete stale files).
6. Start log viewer HTTP server (background goroutine).
7. Initialize Process Supervisor with restart policy config.

### Fase 2 — App Start

1. Validate `APP_BINARY` (exists, regular file, executable).
2. Supervisor builds `exec.Cmd` with `APP_ARGS`, inherited ENV, `APP_WORKDIR`.
3. Attach stdout/stderr pipes to capture module.
4. `Start()` child process; log `application started` with PID.
5. Start health check goroutine if `HEALTHCHECK_ENABLED=true`.

### Fase 3 — Running

1. Capture goroutines drain pipes → append to daily log file.
2. Supervisor blocks on `Wait()` child, signal channel, or health failure.
3. Log viewer serves read-only access to log files.

### Fase 4 — Child Exit / Restart

1. Capture exit code; log `application exited` or `application crashed`.
2. If intentional shutdown → skip restart → Fase 5.
3. Evaluate `APP_RESTART_POLICY` + crash loop guard + `APP_MAX_RESTART`.
4. If restart: wait `APP_RESTART_DELAY`, log `restarting application`, goto Fase 2.
5. If no restart: log `restart skipped`, goto Fase 5.

### Fase 5 — Shutdown

Triggered by: `SIGTERM`/`SIGINT`/`SIGQUIT` (container stop), or policy `never` with no restart.

Sequence:

1. Set intentional shutdown flag (suppress restart).
2. Log `signal received`; forward signal to child.
3. Wait up to `APP_SHUTDOWN_TIMEOUT` for child exit.
4. If timeout: `SIGKILL` child; log `forced shutdown`.
5. Stop health probe and log viewer HTTP server.
6. Flush log buffers, close files.
7. Log `graceful shutdown completed`; exit with appropriate code.

### Fase 6 — Exit

- Policy `never`: propagate child exit code.
- Crash loop / max restart: exit `1`.
- Intentional shutdown: exit `0` (clean stop).

## Signal Forwarding

| Signal ke Launcher | Perilaku MVP |
|--------------------|--------------|
| `SIGTERM` | Forward → child; graceful shutdown path |
| `SIGINT` | Forward → child; graceful shutdown path |
| `SIGQUIT` | Forward → child; graceful shutdown path |
| `SIGKILL` | Cannot catch; kernel kills launcher + child |

## Concurrency Model

- **Single child process** (MVP); supervisor manages one child at a time.
- Log capture: two goroutines (stdout, stderr) with mutex/line buffer.
- Health probe: optional goroutine with ticker.
- Daily log rotation: date check on write or timer.

## Failure Modes

| Kondisi | Perilaku |
|---------|----------|
| Binary missing | Launcher exit `1`, error log |
| Binary not executable | Same |
| LOG_DIR not writable | Exit early before starting app |
| Child OOM killed | Log crash; restart per policy |
| Crash loop detected | Stop restart; launcher exit `1` |
| Viewer port in use | Exit early at init |
| Health check failures | Stop child; restart per policy |

## Batasan Arsitektur MVP

- Satu binary, satu child (supervisor loop).
- No hot reload binary without container restart.
- Log viewer reads filesystem only; no indexing.

## Deploy Layer (host)

Di luar container, `launcher.sh` mengelola lifecycle Docker:

```
Host
  apps/<id>/binary + .env
  launcher.sh ──► docker-compose.yml (generated)
                ──► N containers (1 app each)
                ──► logs/<id>/ per app
```

Setiap container tetap: satu QoLauncher (PID 1) + satu child binary. Detail: [08-docker.md](./08-docker.md).

## Implementability

Desain map langsung ke Go stdlib:

- `os/exec` for child
- `os/signal` for forwarding
- `net/http` for viewer and health probe
- File I/O for logs

Lihat [03-runtime.md](./03-runtime.md) dan [11-process-supervisor.md](./11-process-supervisor.md).
