# Keamanan

Dokumen ini merangkum model threat, kontrol keamanan MVP, dan rekomendasi deployment production untuk QoLauncher.

## Threat Model

| Aset | Risiko | Mitigasi MVP |
|------|--------|--------------|
| Log files (may contain secrets, PII) | Unauthorized read | Basic Auth on viewer |
| Log viewer credentials | Brute force, leak | Strong password, secrets management |
| Child binary | Tampering | Read-only volume mount |
| Container network | Exposed admin port | Firewall, bind internal, reverse proxy |
| Launcher ENV | Password in env dump | Never log secrets |

## Basic Auth

Log viewer menggunakan **HTTP Basic Authentication** (RFC 7617).

### Implementasi

- Compare constant-time for password (`crypto/subtle.ConstantTimeCompare` on hashed or raw bytes).
- Realm: `QoLauncher Log Viewer`.
- All `/logs*` routes require auth except documented health endpoint.

### Keterbatasan

- Credentials sent base64-encoded (not encrypted) over plain HTTP.
- **Wajib** terminate TLS di reverse proxy untuk production.

### Rekomendasi Password

- Min 16 chars random.
- Rotate via container recreate + new secret.
- Jangan commit ke git; use `.env` gitignored or orchestrator secrets.

## Password via ENV

`LOG_PASSWORD` supplied via environment:

```yaml
environment:
  LOG_PASSWORD: ${LOG_PASSWORD}
secrets:
  - log_password
```

Docker Compose secrets map to file; launcher reads ENV injected by entrypoint wrapper (future) or direct ENV from orchestrator.

**Launcher MUST NOT:**

- Print `LOG_PASSWORD` in logs.
- Include in `/health` or `/logs` responses.
- Pass to child process (strip from inherited env if duplicated).

## Exposure Log Tanpa Autentikasi

**Policy:** Tidak expose isi log tanpa autentikasi.

| Surface | Auth MVP |
|---------|----------|
| `GET /logs` | Yes |
| `GET /logs/{file}` | Yes |
| `GET /logs/{file}/download` | Yes |
| `GET /health` | No (metadata only) |
| Raw `LOG_DIR` volume | OS-level (out of scope) |

Disable viewer entirely (`VIEWER_ENABLED=false`) if no network access to logs needed; access via volume only.

## HTTPS Reverse Proxy

QoLauncher MVP serves HTTP only inside container. Production pattern:

```
Internet ──TLS──► Nginx / Traefik / Caddy
                      │
                      ├── :443/api ──► app:APP_PORT
                      └── :443/logs ──► app:LOG_PORT (with auth at proxy optional double layer)
```

### Nginx example (snippet)

```nginx
server {
    listen 443 ssl;
    server_name logs.example.com;

    ssl_certificate     /etc/ssl/certs/fullchain.pem;
    ssl_certificate_key /etc/ssl/private/privkey.pem;

    location / {
        proxy_pass http://qolauncher:8081;
        proxy_set_header Host $host;
        # Optional: additional auth at proxy
        # auth_basic "Logs";
        # auth_basic_user_file /etc/nginx/.htpasswd;
    }
}
```

Double auth (proxy + launcher) acceptable; single strong layer minimum.

## Pembatasan IP (Future — V2)

Reserved config `VIEWER_ALLOW_CIDRS`:

```bash
VIEWER_ALLOW_CIDRS=10.0.0.0/8,192.168.0.0/16
```

Middleware rejects other IPs before auth (reduce brute force surface).

Not MVP; document for roadmap alignment.

## Container Hardening

Rekomendasi Dockerfile (see [08-docker.md](./08-docker.md)):

| Practice | Reason |
|----------|--------|
| Non-root user | Limit container breakout impact |
| Read-only rootfs where possible | Immutability |
| Read-only binary mount | Prevent runtime tampering |
| Drop capabilities | Minimal privileges |
| No `--privileged` | — |

## Child Process Isolation

Launcher runs child as same UID/GID as launcher (MVP). Future: `APP_USER` drop privileges before exec.

## Supply Chain

- Publish image with digest pinning recommendation.
- Sign releases (cosign) — V2 roadmap.

## Audit Checklist Production

- [ ] `LOG_PASSWORD` strong & from secret store
- [ ] `LOG_PORT` not published to public internet directly
- [ ] TLS termination on reverse proxy
- [ ] Log volume permissions restricted
- [ ] Binary mounted `:ro`
- [ ] `docker inspect` does not expose password in plain labels

## Referensi

- Viewer endpoints: [06-log-viewer.md](./06-log-viewer.md)
- Supervisor: [11-process-supervisor.md](./11-process-supervisor.md)
- Docker: [08-docker.md](./08-docker.md)
- Config: [04-configuration.md](./04-configuration.md)
