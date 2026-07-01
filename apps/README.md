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

Contoh bawaan:

| App | APP | Viewer |
|-----|-----|--------|
| `http-server` | :8080 | :8081 |
| `hello` | — | :8082 |

## Build binary untuk Docker

```bash
# App kamu sendiri
GOOS=linux GOARCH=amd64 go build -o apps/my-api/server .

# Contoh bawaan repo
make build-examples
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
| `http-server` | http://localhost:8081/logs | Daftar file log harian app ini |
| `hello` | http://localhost:8082/logs | Daftar file log harian app ini |

File log di host: `logs/<app-id>/` (mis. `logs/http-server/2026-07-01.log`).

Untuk ganti app → buka viewer di port app tersebut. Homepage viewer menampilkan **file log per tanggal**, bukan picker folder multi-app.
