# Runtime Behavior

Dokumen ini menjelaskan perilaku QoLauncher dari startup hingga exit. Process Supervisor detail: [11-process-supervisor.md](./11-process-supervisor.md). Konfigurasi: [04-configuration.md](./04-configuration.md).

## Startup Sequence

```
main()
  ├─ parse CLI flags (optional overrides)
  ├─ load Config from ENV + CLI
  ├─ validate Config
  ├─ setup launcher logger (stderr, level=LOG_LEVEL)
  ├─ log: launcher started
  ├─ init LogWriter(LOG_DIR, TZ)
  ├─ run retention sweep (LOG_RETENTION_DAYS)
  ├─ start LogViewer(LOG_PORT, auth)
  └─ Supervisor.Run(AppSpec)  ← blocks until launcher exit
        ├─ start child
        ├─ optional health probe goroutine
        └─ restart loop per policy
```

Startup **fail-fast**: kesalahan konfigurasi atau filesystem tidak melanjutkan ke app start.

## Membaca ENV

Prioritas konfigurasi:

1. Default internal (code)
2. Environment variables
3. CLI flags (highest priority)

ENV child process: launcher meneruskan environment container kecuali variabel reserved (prefiks `LAUNCHER_`, lihat [04-configuration.md](./04-configuration.md)).

## Validasi

### Konfigurasi

| Field | Rule |
|-------|------|
| `APP_BINARY` | Required; non-empty path |
| `APP_RESTART_POLICY` | `never`, `on-failure`, or `always` |
| `APP_PORT` | Optional; integer 0 or 1–65535 |
| `LOG_PORT` | Integer 1–65535 (default 8081) |
| `LOG_DIR` | Directory or creatable |
| `LOG_RETENTION_DAYS` | >= 0 |
| `APP_RESTART_BURST` | >= 1 |
| `LOG_USERNAME` / `LOG_PASSWORD` | Required if `VIEWER_ENABLED=true` |
| `HEALTHCHECK_URL` | Required if `HEALTHCHECK_ENABLED=true` |

### Binary

1. `os.Stat` — exists, not directory.
2. Permission check: executable or warn and try exec.
3. Optional: ELF magic for friendly error.

Validation errors → log + exit `1`.

## Menjalankan Binary

Supervisor membangun dan start child:

```go
cmd := exec.Command(appBinary, appArgs...)
cmd.Dir = appWorkdir
cmd.Env = os.Environ()
cmd.Stdout = stdoutPipe
cmd.Stderr = stderrPipe
err := cmd.Start()
```

Launcher tidak memodifikasi binary.

## Monitoring Process

Supervisor loop:

1. `cmd.Wait()` until child exit.
2. Parallel: capture goroutines, signal handler, health probe.
3. On exit: evaluate restart policy (see [11-process-supervisor.md](./11-process-supervisor.md)).

When child exits without restart:

1. Close pipes (EOF capture).
2. Flush log writer.
3. Launcher exits with appropriate code.

## Exit Code Propagation

| Scenario | Container exit code |
|----------|---------------------|
| Policy `never`, child exit 0 | 0 |
| Policy `never`, child exit N | N |
| Policy `never`, child signaled | 128 + S |
| Intentional shutdown, clean | 0 |
| Intentional shutdown, forced kill | 137 |
| Crash loop guard triggered | 1 |
| `APP_MAX_RESTART` exceeded | 1 |
| Launcher init failure | 1 |

Saat restart policy aktif, container tetap running selama supervisor loop.

## Signal Forwarding

1. Register `signal.Notify` on `SIGTERM`, `SIGINT`, `SIGQUIT`.
2. On signal: set intentional shutdown, forward to child.
3. Wait `APP_SHUTDOWN_TIMEOUT`; then `SIGKILL` if needed.

Detail sequence: [11-process-supervisor.md](./11-process-supervisor.md).

## Restart Policy

MVP implements full restart policy via Process Supervisor.

| Policy | Child exit 0 | Child exit ≠ 0 |
|--------|--------------|----------------|
| `never` | Launcher exit 0 | Launcher exit N |
| `on-failure` | Launcher exit 0 | Restart after delay |
| `always` | Restart | Restart |

Guards: `APP_RESTART_BURST` + `APP_RESTART_WINDOW`, `APP_MAX_RESTART`.

Intentional shutdown never triggers restart.

## Health Check (Optional)

If `HEALTHCHECK_ENABLED=true`:

- Probe goroutine HTTP GET `HEALTHCHECK_URL` every `HEALTHCHECK_INTERVAL`.
- `HEALTHCHECK_FAILURES` consecutive failures → stop child → restart per policy.

Default disabled. See [11-process-supervisor.md](./11-process-supervisor.md).

## Stdout/Stderr Capture

- Each line prefixed with timestamp + stream tag ([05-logging.md](./05-logging.md)).
- Serialized writes; pipe backpressure applies.
- Persists across restarts (same daily file).

## Log Rotation Interaction

Startup + periodic sweep; no app restart required.

## Log Viewer Runtime

Background HTTP on `LOG_PORT`; independent of app; stopped during shutdown.

## Observability (Launcher / Supervisor)

Supervisor events to **stderr** (not app log file):

```
2026-07-01T10:00:00Z level=info msg="launcher started" version=0.1.0
2026-07-01T10:00:00Z level=info msg="application started" binary=/app/server pid=42 restart_count=0
2026-07-01T10:05:00Z level=warn msg="application crashed" pid=42 exit_code=2
2026-07-01T10:05:00Z level=info msg="restarting application" delay=3s policy=on-failure
```

Full event catalog: [11-process-supervisor.md](./11-process-supervisor.md).

## State Diagram

```
        ┌──────────┐
        │  INIT    │
        └────┬─────┘
             │
             ▼
        ┌──────────┐
   ┌───►│ RUNNING  │◄────────────┐
   │    └────┬─────┘             │
   │         │                   │ restart
   │    child exit /             │ (after delay)
   │    health fail              │
   │         ▼                   │
   │    ┌──────────┐             │
   │    │ EVALUATE │─────────────┘
   │    └────┬─────┘
   │         │ no restart
   │    signal ▼
   │    ┌──────────┐
   └─── │ SHUTDOWN │
        └────┬─────┘
             ▼
        ┌──────────┐
        │  EXIT    │
        └──────────┘
```

## Edge Cases

| Case | Handling |
|------|----------|
| App exits before viewer ready | Viewer started in init before supervisor |
| Restart during shutdown | Blocked by intentional shutdown flag |
| `APP_ARGS` empty | Start with no args |
| Health check on non-HTTP app | Keep `HEALTHCHECK_ENABLED=false` |
| Port conflict LOG_PORT = APP_PORT | Fail validation at startup |

## Referensi

- Supervisor: [11-process-supervisor.md](./11-process-supervisor.md)
- Konfigurasi: [04-configuration.md](./04-configuration.md)
- Format log: [05-logging.md](./05-logging.md)
- Viewer: [06-log-viewer.md](./06-log-viewer.md)
