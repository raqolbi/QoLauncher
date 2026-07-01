# Process Supervisor

QoLauncher bertindak sebagai **Process Supervisor** dan **PID 1** di dalam container Docker. Modul ini mengelola lifecycle binary Go yang dijalankan sebagai child process — tanpa memerlukan perubahan source code aplikasi.

Dokumen ini adalah spesifikasi desain supervisor untuk MVP. Implementasi merujuk modul `supervisor` (lihat [02-architecture.md](./02-architecture.md)).

## Tujuan

Supervisor harus mampu:

| # | Capability |
|---|------------|
| 1 | Menjalankan binary Go sebagai child process |
| 2 | Memonitor status process (running, exited, signaled) |
| 3 | Menangkap exit code dan signal termination |
| 4 | Restart otomatis sesuai `APP_RESTART_POLICY` |
| 5 | Meneruskan signal (`SIGTERM`, `SIGINT`, `SIGQUIT`) dari Docker/OS ke child |
| 6 | Graceful shutdown dengan timeout sebelum `SIGKILL` |
| 7 | Mencatat seluruh lifecycle event ke log launcher (stderr) |

## Posisi dalam Arsitektur

```
Docker (SIGTERM on stop)
  │
  ▼
QoLauncher PID 1
  │
  ├── Config Loader
  ├── Log Viewer (goroutine)
  ├── Log Writer / Rotator
  │
  └── Process Supervisor ◄── dokumen ini
        ├── start / stop / restart child
        ├── signal forwarding
        ├── restart policy engine
        ├── crash loop guard
        └── health check probe (optional)
              │
              ▼
        Go Application Binary (child)
```

Supervisor **tidak** menggantikan orchestrator Docker/Kubernetes; ia mengelola **satu** child process di dalam satu container.

---

## Restart Policy

Dikonfigurasi via ENV. Lihat [04-configuration.md](./04-configuration.md) untuk referensi lengkap.

### ENV

```bash
APP_RESTART_POLICY=always
APP_RESTART_DELAY=3s
APP_MAX_RESTART=0
APP_RESTART_WINDOW=60s
APP_RESTART_BURST=10
APP_SHUTDOWN_TIMEOUT=30s
```

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_RESTART_POLICY` | `never` | `never` \| `on-failure` \| `always` |
| `APP_RESTART_DELAY` | `3s` | Tunggu sebelum start ulang setelah exit |
| `APP_MAX_RESTART` | `0` | Max total restart seumur hidup launcher; `0` = unlimited |
| `APP_RESTART_WINDOW` | `60s` | Sliding window untuk crash loop detection |
| `APP_RESTART_BURST` | `10` | Max restart dalam window sebelum launcher fatal exit |
| `APP_SHUTDOWN_TIMEOUT` | `30s` | Tunggu graceful stop child sebelum `SIGKILL` |

### `never`

- Child exit → launcher **tidak** restart.
- Launcher exit dengan exit code child (atau 128+signal).
- Cocok untuk batch job, migration, one-shot task.

```
App exit 0  → Launcher exit 0
App exit 42 → Launcher exit 42
```

### `on-failure`

- Restart hanya jika exit code **≠ 0** atau terminated by signal (kecuali shutdown intentional).
- Exit code **0** → launcher exit 0, tidak restart.

```
App exit 0  → Launcher exit 0
App exit 42 → wait APP_RESTART_DELAY → restart
App crash loop → crash loop guard → Launcher exit 1
```

### `always`

- Restart setiap kali child exit, **termasuk** exit 0.
- Launcher hanya exit saat: intentional shutdown, crash loop guard, atau `APP_MAX_RESTART` tercapai.

```
App exit 0  → wait APP_RESTART_DELAY → restart
App exit 42 → wait APP_RESTART_DELAY → restart
docker stop → intentional shutdown → Launcher exit 0 (no restart)
```

### Intentional Shutdown vs Crash

| Event | Restart? |
|-------|----------|
| Child exit spontaneous | Sesuai policy |
| Launcher receives `SIGTERM`/`SIGINT`/`SIGQUIT` (container stop) | **No** — graceful shutdown path |
| Health check failure threshold | **Yes** — treated as failure, follow policy |
| Crash loop guard triggered | **No** — launcher exits error |

Intentional shutdown flag diset saat launcher menerima stop signal; supervisor tidak restart meskipun policy `always`.

### `APP_RESTART_DELAY`

Pause antara child exit dan start berikutnya. Mencegah tight restart loop dan memberi waktu resource cleanup (port release, file handles).

### `APP_MAX_RESTART`

Counter total restart sejak launcher start.

| Value | Behavior |
|-------|----------|
| `0` | Unlimited (subject to crash loop guard only) |
| `N > 0` | After N restarts, log `maximum restart reached`, launcher exit `1` |

Counter **tidak** increment saat first start — hanya restart subsequent.

---

## Crash Loop Protection

Mencegah container infinite restart saat aplikasi broken.

### Mekanisme

Sliding window `APP_RESTART_WINDOW`:

1. Setiap restart, catat timestamp ke ring buffer / slice.
2. Sebelum restart berikutnya, hitung restart dalam window `[now - APP_RESTART_WINDOW, now]`.
3. Jika count ≥ `APP_RESTART_BURST` → **stop restarting**, launcher exit `1`.

### Contoh Default

```bash
APP_RESTART_BURST=10
APP_RESTART_WINDOW=60s
```

> Jika aplikasi crash lebih dari **10 kali** dalam **60 detik**, launcher berhenti restart dan keluar dengan status error.

### Log Event

```
level=error msg="crash loop detected" restarts_in_window=10 window=60s
level=error msg="maximum restart reached" reason=crash_loop
```

### Interaksi dengan `APP_MAX_RESTART`

| Guard | Scope |
|-------|-------|
| Crash loop (`APP_RESTART_BURST` + `APP_RESTART_WINDOW`) | Short-term burst |
| `APP_MAX_RESTART` | Lifetime total cap |

Keduanya independen; whichever triggers first wins.

---

## Exit Code

### Interpretasi

| Exit Code | Interpretasi | Restart (`on-failure`) | Restart (`always`) |
|-----------|--------------|------------------------|---------------------|
| `0` | Normal completion | No | Yes |
| `1–255` | Failure | Yes | Yes |
| Signaled (e.g. SIGSEGV) | Crash | Yes | Yes |
| `128 + N` | Killed by signal N | Yes* | Yes* |

\*Kecuali signal dari intentional shutdown path (launcher sent SIGTERM then child exited).

### Propagasi ke Container

| Scenario | Container exit code |
|----------|---------------------|
| Policy `never`, child exit N | N |
| Policy `never`, child exit 0 | 0 |
| Intentional shutdown, child stopped cleanly | 0 |
| Intentional shutdown, child killed after timeout | 137 atau child state |
| Crash loop guard | `1` |
| `APP_MAX_RESTART` exceeded | `1` |
| Launcher init failure | `1` |

Saat policy restart aktif dan child masih dalam loop restart, launcher **belum** exit — container tetap running.

---

## Signal Forwarding

Launcher menerima signal dari Docker (via `docker stop` → `SIGTERM`) dan meneruskannya ke child.

### Signal MVP

| Signal | Sumber | Perilaku Launcher |
|--------|--------|-------------------|
| `SIGTERM` | `docker stop`, orchestrator | Forward ke child → graceful shutdown path |
| `SIGINT` | Ctrl+C (local dev) | Forward ke child → graceful shutdown path |
| `SIGQUIT` | Manual / debug | Forward ke child → graceful shutdown path |
| `SIGKILL` | `docker kill -9` | Tidak dapat ditangkap; kernel kill all |

### Graceful Shutdown Sequence

```
Signal received (SIGTERM/SIGINT/SIGQUIT)
  │
  ├─ log: signal received
  ├─ set intentional_shutdown = true
  ├─ forward signal → child process
  │
  ▼
Wait up to APP_SHUTDOWN_TIMEOUT
  │
  ├── child exited → log: graceful shutdown completed → stop viewer → launcher exit 0
  │
  └── timeout → SIGKILL child → log: forced shutdown → launcher exit 137
```

Child application **harus** menangkap signal di aplikasi Go jika ingin graceful HTTP drain — QoLauncher hanya forward; tidak inject handler.

Contoh aplikasi (opsional, di sisi developer):

```go
// Tidak wajib — QoLauncher tetap jalan tanpa ini
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
<-quit
// drain connections...
```

### `APP_SHUTDOWN_TIMEOUT`

Harus < Docker `stop_grace_period` (buffer ~5s). Lihat [08-docker.md](./08-docker.md).

---

## Health Check (Opsional MVP)

Fitur opsional; aktif hanya jika `HEALTHCHECK_ENABLED=true`. Desain saja — implementasi MVP boleh minimal (HTTP probe).

### ENV

```bash
HEALTHCHECK_ENABLED=true
HEALTHCHECK_TYPE=http
HEALTHCHECK_URL=http://127.0.0.1:8080/health
HEALTHCHECK_INTERVAL=30s
HEALTHCHECK_TIMEOUT=5s
HEALTHCHECK_FAILURES=3
```

| Variable | Default | Description |
|----------|---------|-------------|
| `HEALTHCHECK_ENABLED` | `false` | Enable active probing |
| `HEALTHCHECK_TYPE` | `http` | MVP: `http` only; future: `tcp` |
| `HEALTHCHECK_URL` | — | Required if enabled; full URL |
| `HEALTHCHECK_INTERVAL` | `30s` | Probe interval |
| `HEALTHCHECK_TIMEOUT` | `5s` | Per-probe timeout |
| `HEALTHCHECK_FAILURES` | `3` | Consecutive failures before action |

### Perilaku

1. Goroutine probe setiap `HEALTHCHECK_INTERVAL`.
2. HTTP GET `HEALTHCHECK_URL`; success = status 2xx within timeout.
3. Consecutive failures ≥ `HEALTHCHECK_FAILURES`:
   - Log `health check failed`
   - Stop child (`SIGTERM` → wait → `SIGKILL`)
   - Treat as failure → apply `APP_RESTART_POLICY`
4. Success resets failure counter.

Health check failure **bukan** intentional shutdown — restart policy applies.

### Non-HTTP Apps

Set `HEALTHCHECK_ENABLED=false` (default). Probe tidak jalan.

---

## Logging Supervisor Events

Semua event supervisor ditulis ke **stderr launcher** (structured log), **bukan** ke file app log. Format konsisten dengan [03-runtime.md](./03-runtime.md).

### Event Catalog

| Event | Level | Fields |
|-------|-------|--------|
| `launcher started` | info | version, config summary |
| `application started` | info | binary, pid, restart_count |
| `application stopped` | info | pid, reason |
| `application exited` | info | pid, exit_code, signal |
| `application crashed` | warn | pid, exit_code, signal |
| `restarting application` | info | delay, restart_count, policy |
| `restart skipped` | info | reason, exit_code, policy |
| `maximum restart reached` | error | restart_count, limit |
| `crash loop detected` | error | restarts_in_window, window |
| `signal received` | info | signal |
| `graceful shutdown completed` | info | pid, duration |
| `forced shutdown` | warn | pid, timeout |
| `health check failed` | warn | url, consecutive_failures |

### Contoh Output

```
2026-07-01T10:00:00Z level=info msg="launcher started" version=0.1.0
2026-07-01T10:00:00Z level=info msg="application started" binary=/app/server pid=42 restart_count=0
2026-07-01T10:05:00Z level=warn msg="application crashed" pid=42 exit_code=2
2026-07-01T10:05:00Z level=info msg="restarting application" delay=3s restart_count=1 policy=on-failure
2026-07-01T10:05:03Z level=info msg="application started" binary=/app/server pid=58 restart_count=1
2026-07-01T10:10:00Z level=info msg="signal received" signal=SIGTERM
2026-07-01T10:10:02Z level=info msg="graceful shutdown completed" pid=58 duration=2.1s
```

Event `application crashed` vs `application exited`:

- **crashed** — exit_code ≠ 0 atau killed by signal (non-shutdown).
- **exited** — exit_code = 0.

---

## Diagram Lifecycle

### State Machine

```
                    ┌─────────────────┐
                    │ Launcher Start  │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   Load Config   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
              ┌────►│ Start Application│◄────┐
              │     └────────┬────────┘     │
              │              │               │
              │              ▼               │
              │     ┌─────────────────┐      │
              │     │     Running     │      │
              │     └────────┬────────┘      │
              │              │               │
              │     ┌────────┴────────┐      │
              │     │                 │      │
              │  Signal?          Child exit? │
              │     │                 │      │
              │     ▼                 ▼      │
              │  Shutdown         Evaluate   │
              │  (no restart)     exit code  │
              │     │                 │      │
              │     │          ┌──────┴──────┐│
              │     │          │             ││
              │     │       exit 0       exit ≠0
              │     │          │             ││
              │     │     Policy check   Policy check
              │     │          │             ││
              │     │     ┌────┴────┐   ┌────┴────┐
              │     │     │         │   │         │
              │     │  restart?  stop restart? stop
              │     │     │         │   │         │
              │     │     │    Stopped     │    Stopped
              │     │     │    (exit 0)    │    (exit N)
              │     │     │                │
              │     │     ▼                ▼
              │     │  Crash loop?    Restart Policy
              │     │     │                │
              │     │  yes  │ no        never → Stopped
              │     │     ▼                │
              │     │  Stopped             on-failure / always
              │     │  (exit 1)                 │
              │     │                           ▼
              │     │                    APP_RESTART_DELAY
              │     │                           │
              └─────┴───────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │     Stopped     │
                    │ (launcher exit) │
                    └─────────────────┘
```

### Alur Sederhana (Crash Path)

```
Launcher Start
      ↓
Load Config
      ↓
Start Application
      ↓
   Running
      ↓
   Crash? ──No──► Running (loop)
      │
     Yes
      ↓
Restart Policy?
      │
      ├── never ──────────► Stopped (exit N)
      │
      └── on-failure / always
              ↓
        Crash loop guard OK?
              │
              ├── No ──► Stopped (exit 1)
              │
              └── Yes
                    ↓
              APP_RESTART_DELAY
                    ↓
              Start Application
                    ↓
                 Running
```

---

## Modularitas

Supervisor diimplementasikan sebagai package terpisah dengan interface jelas:

```go
type Supervisor interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    State() ProcessState
}
```

Komponen inti (config, logging, viewer) **tidak** diubah saat menambah fitur supervisor — supervisor hanya consume `Config` dan emit events ke logger.

Future (V3): multi-app = multiple supervisor instances dari manifest; core tetap sama.

---

## Referensi

- Arsitektur: [02-architecture.md](./02-architecture.md)
- Runtime: [03-runtime.md](./03-runtime.md)
- Konfigurasi ENV: [04-configuration.md](./04-configuration.md)
- Docker grace period: [08-docker.md](./08-docker.md)
- Roadmap: [10-roadmap.md](./10-roadmap.md)
