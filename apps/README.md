# Folder `apps/` — taruh binary Go kamu di sini

> **Contoh bawaan:** Repo ini sudah berisi demo `http-server` dan `hello` (source + `.env`).
> Kamu **boleh hapus semua isi folder ini** dan ganti dengan app sendiri — cukup copy binary hasil `go build` + optional `.env`.

## Struktur per app (disarankan, multi-app)

```
apps/
  my-api/
    server          ← binary (GOOS=linux)
    .env            ← wajib beda port jika jalan bersamaan
  worker/
    worker
    .env
```

## Multi-app bersamaan

`launcher.sh` bisa menjalankan **beberapa app sekaligus** — masing-masing jadi container Docker terpisah.

Setiap app **harus punya port unik** di `.env`-nya:

| Variable | Keterangan |
|----------|------------|
| `APP_PORT` | Port aplikasi (`0` jika tidak expose HTTP) |
| `LOG_PORT` | Port log viewer (harus unik per app) |

Contoh bawaan (lihat `apps/*/.env`):

| App | APP | Viewer |
|-----|-----|--------|
| `http-server` | :9998 | :9999 |
| `hello` | — (tidak HTTP) | :9997 |

## Build binary untuk Docker

Image QoLauncher memakai **Alpine (musl)**. Binary harus **static** (`CGO_ENABLED=0`), bukan dynamic glibc dari host — kalau tidak, container restart loop dan log cuma `launcher started` tanpa `application started`.

```bash
# App kamu sendiri (static binary untuk image Alpine)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o apps/my-api/server .

# Contoh bawaan repo
make build-examples

# Verifikasi
file apps/my-api/server   # harus: statically linked
```

## Menjalankan

```bash
./launcher.sh
```

Pilih **Run** / **Stop** / **Restart** / **Logs** → ketik:

- `all` — semua app
- `2` — satu app
- `1,3` — beberapa app

## Log viewer & file log (per app)

Setiap app punya **viewer terpisah** (port berbeda), bukan satu halaman dengan pilihan folder:

| App | Viewer | Isi homepage `/logs` |
|-----|--------|----------------------|
| `http-server` | http://localhost:9999/logs | Daftar file log harian app ini |
| `hello` | http://localhost:9997/logs | Daftar file log harian app ini |

File log di host: `logs/<app-id>/` (mis. `logs/hello/2026-07-01.log`).

### Output app vs log launcher

| Sumber | Contoh | Di mana |
|--------|--------|---------|
| **Launcher** (supervisor) | `launcher started`, `application exited` | `docker compose logs` |
| **Aplikasi** (stdout/stderr) | `Hello from QoLauncher`, `listening on :9998` | `logs/<app-id>/YYYY-MM-DD.log` dan log viewer |

Format baris app: `{timestamp} [stdout] {message}` — lihat [docs/05-logging.md](../docs/05-logging.md).

Untuk ganti app → buka viewer di port `LOG_PORT` app tersebut. Homepage viewer menampilkan **file log per tanggal**, bukan picker folder multi-app.
