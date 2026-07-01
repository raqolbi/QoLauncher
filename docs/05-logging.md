# Sistem Logging

QoLauncher menangkap **stdout** dan **stderr** child process ke file log harian di `LOG_DIR`, dengan rotasi berdasarkan retensi.

## Persyaratan

| # | Requirement | MVP |
|---|-------------|-----|
| 1 | Capture stdout aplikasi | Yes |
| 2 | Capture stderr aplikasi | Yes |
| 3 | Log harian (satu file per hari) | Yes |
| 4 | Nama file jelas, sortable | Yes |
| 5 | Hapus file > X hari | Yes |
| 6 | Gzip archive | V2 |
| 7 | Structured JSON | No (plain text) |

## Alur Capture

```
Go Binary
  ├─ stdout ──► pipe ──► capture goroutine ──┐
  └─ stderr ──► pipe ──► capture goroutine ──┼──► LogWriter ──► YYYY-MM-DD.log
                                            │
Launcher internal logs ──► stderr (container)  (NOT captured)
```

Launcher **tidak** men-capture log internal sendiri ke file app log (hindari noise & recursion).

## Penamaan File

Satu file per hari kalender menurut `TZ`:

```
LOG_DIR/
  2026-06-29.log
  2026-06-30.log
  2026-07-01.log
```

Pattern: `{YYYY-MM-DD}.log`

- Hanya karakter aman; tidak ada spasi.
- Saat rollover midnight, writer switch ke file baru tanpa restart app.
- File dibuka append (`O_APPEND|O_CREATE|O_WRONLY`).

## Format Baris Log

Setiap baris dari stdout/stderr dinormalisasi:

```
{timestamp} [{stream}] {message}
```

| Field | Format | Contoh |
|-------|--------|--------|
| `timestamp` | RFC3339 dengan offset TZ | `2026-07-01T15:04:05+07:00` |
| `stream` | `stdout` atau `stderr` | `[stderr]` |
| `message` | Raw bytes line dari app (trim `\n`) | `listen tcp :8080` |

### Contoh

```
2026-07-01T10:00:01+07:00 [stdout] Server starting on :8080
2026-07-01T10:00:02+07:00 [stderr] warning: deprecated option
2026-07-01T10:05:00+07:00 [stdout] GET /health 200 1ms
```

### Partial Lines

Jika aplikasi menulis tanpa trailing newline, buffer sampai newline atau flush on EOF (child exit).

### Binary / Non-UTF8

Tulis as-is; viewer menampilkan dengan content-type `text/plain; charset=utf-8`, invalid sequences replaced (viewer concern).

## Rotasi

### Daily Rollover

Trigger:

- Check current date setiap write **or** background ticker (e.g. 1m).

On date change:

1. Close current file handle.
2. Open `{new-date}.log`.

Tidak truncate file lama.

### Retention Delete

Jalankan:

- Saat launcher startup.
- Periodic (e.g. setiap 24h).

Algorithm:

```
cutoff = today(TZ) - LOG_RETENTION_DAYS
for each file matching YYYY-MM-DD.log in LOG_DIR:
    if file_date < cutoff: os.Remove(file)
```

`LOG_RETENTION_DAYS=0` → skip deletion.

Future V2: gzip `*.log` older than N days before delete policy applies to `.log.gz`.

## Concurrency & Durability

- Mutex around file write untuk atomic lines.
- `Sync()` optional on interval (V2); MVP rely on OS buffer + graceful shutdown flush.
- Shutdown: flush buffers, close file.

## Kapasitas

- Tidak ada max file size MVP; OS filesystem limits apply.
- High throughput: line-buffered mutex may bottleneck; acceptable MVP for typical web apps.

## Perbedaan dengan Launcher Logs

| Aspek | App log (file) | Launcher log (stderr) |
|-------|----------------|------------------------|
| Sumber | Child stdout/stderr | Launcher + supervisor internals |
| Format | `{ts} [stream] msg` | Structured key=value |
| Viewer | Yes | No (use `docker logs`) |
| Retention | LOG_RETENTION_DAYS | Docker logging driver |
| Event types | App output lines | startup, crash, restart, shutdown — lihat [11-process-supervisor.md](./11-process-supervisor.md) |

## Integrasi Docker

`docker logs` tetap menampilkan stderr launcher + optionally forwarded child if configured; MVP **child output tidak duplicate** ke launcher stderr — hanya ke file. Untuk live tail container-level, mount `LOG_DIR` or use viewer.

**Multi-app (`launcher.sh`):** setiap container mount `logs/<app-id>/` di host ke `LOG_DIR` container — log file terisolasi per app.

Future V2 realtime tail viewer reads file or ring buffer.

## Referensi

- Config: [04-configuration.md](./04-configuration.md)
- Viewer: [06-log-viewer.md](./06-log-viewer.md)
