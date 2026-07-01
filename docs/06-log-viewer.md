# Log Viewer

Web UI minimal untuk browsing dan download log harian yang ditulis QoLauncher ke `LOG_DIR`.

## Fitur

### MVP

| Fitur | Deskripsi |
|-------|-----------|
| Basic Auth | Semua endpoint protected |
| Daftar file log | List `*.log` sorted newest first |
| Lihat isi file | Raw text display |
| Download file | `Content-Disposition: attachment` |

### Future (V2+)

| Fitur | Milestone |
|-------|-----------|
| Search (full-text / filter) | V2 |
| Realtime tail | V2 |
| Highlight `ERROR` / level | V2 |
| Pagination | V2 |
| WebSocket stream | V2 |

## Arsitektur

```
Browser ──HTTPS──► Reverse Proxy (optional)
                      │
                      ▼ HTTP
                 Log Viewer :LOG_PORT
                      │
                      ├─ Auth middleware
                      ├─ File list handler
                      ├─ File read handler
                      └─ Download handler
                      │
                      ▼
                 LOG_DIR/*.log (read-only)
```

Viewer **read-only**; tidak edit/delete via HTTP.

## Multi-app (via `launcher.sh`)

Saat menjalankan beberapa app, **setiap container punya viewer sendiri** — bukan satu portal dengan picker folder.

| App | Host log dir | Viewer URL |
|-----|--------------|------------|
| `http-server` | `logs/http-server/` | `http://localhost:8081/logs` |
| `hello` | `logs/hello/` | `http://localhost:8082/logs` |

- Homepage `/logs` menampilkan **daftar file log harian** (`YYYY-MM-DD.log`) untuk app container tersebut saja.
- Ganti app → buka viewer di **port `LOG_PORT`** app tersebut (set di `apps/<id>/.env`).
- Unified dashboard multi-app (satu port, pilih folder) **belum** ada — direncanakan V3 ([10-roadmap.md](./10-roadmap.md)).

## Autentikasi

HTTP Basic Auth (`Authorization: Basic base64(user:pass)`).

Credentials dari `LOG_USERNAME` / `LOG_PASSWORD`.

401 response jika missing/invalid:

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="QoLauncher Log Viewer"
```

## Endpoint HTTP

Base URL: `http://<host>:<LOG_PORT>`

### `GET /`

Landing / redirect ke `/logs`.

**Response:** `302 Location: /logs` atau simple HTML index.

---

### `GET /logs`

Daftar file log.

**Auth:** Required

**Response:** `200 application/json`

```json
{
  "logs": [
    {
      "name": "2026-07-01.log",
      "date": "2026-07-01",
      "size_bytes": 1048576,
      "modified_at": "2026-07-01T23:59:00Z"
    },
    {
      "name": "2026-06-30.log",
      "date": "2026-06-30",
      "size_bytes": 524288,
      "modified_at": "2026-06-30T23:58:00Z"
    }
  ]
}
```

Sorting: `date` descending.

**Errors:**

| Code | Condition |
|------|-----------|
| 401 | Unauthorized |
| 500 | Cannot read LOG_DIR |

---

### `GET /logs/{filename}`

Lihat isi file log.

**Auth:** Required

**Path param:** `filename` — must match `^\d{4}-\d{2}-\d{2}\.log$` (path traversal blocked).

**Query params (MVP):**

| Param | Default | Description |
|-------|---------|-------------|
| `format` | `html` | `html` or `raw` |

**Response `format=raw`:** `200 text/plain`

Body: raw file content.

**Response `format=html`:** `200 text/html`

Simple page: `<pre>` wrapped content + link download.

**Errors:**

| Code | Condition |
|------|-----------|
| 400 | Invalid filename |
| 401 | Unauthorized |
| 404 | File not found |
| 413 | File too large (future limit) |

MVP: no size limit; document recommended max ~50MB for browser view.

---

### `GET /logs/{filename}/download`

Download file log.

**Auth:** Required

**Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/octet-stream
Content-Disposition: attachment; filename="2026-07-01.log"
```

Body: file bytes.

---

### `GET /health`

Health check viewer (launcher-level, not app).

**Auth:** MVP none (internal use); V2 optional auth or separate port.

**Response:** `200 application/json`

```json
{
  "status": "ok",
  "viewer": "enabled"
}
```

Used by Docker HEALTHCHECK for launcher/viewer process only.

> **Catatan:** Ini berbeda dari `HEALTHCHECK_URL` yang dipakai Process Supervisor untuk probe **aplikasi**. Lihat [11-process-supervisor.md](./11-process-supervisor.md).

---

## Future Endpoints (V2)

### `GET /logs/{filename}/tail`

SSE or WebSocket stream of new lines.

Query: `from_offset`, `follow=true`.

### `GET /logs/search`

Query: `q=error
&file=2026-07-01.log
&limit=100`.

### `GET /logs/{filename}` pagination

Query: `offset`, `limit` for large files.

## UI MVP (HTML)

Minimal embedded templates, no JS framework:

- `/logs` — table: Date, Size, Actions (View, Download)
- `/logs/{filename}?format=html` — monospace `<pre>`, nav back

Styling: minimal inline CSS, readable dark/light neutral.

## Security Notes

- Path traversal prevention mandatory (`..`, absolute paths rejected).
- Only files matching log pattern listed/served.
- Rate limiting: future (V2).
- See [07-security.md](./07-security.md).

## Disable Viewer

`VIEWER_ENABLED=false` → no HTTP server; endpoints unavailable.

## Port Conflict

If `LOG_PORT` equals `APP_PORT` and both bind same interface, startup fails validation with clear error.

## Referensi

- Auth: [07-security.md](./07-security.md)
- Log format: [05-logging.md](./05-logging.md)
- Config: [04-configuration.md](./04-configuration.md)
- Deploy multi-app: [08-docker.md](./08-docker.md)
