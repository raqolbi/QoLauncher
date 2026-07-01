# Roadmap

Roadmap QoLauncher dibagi per milestone. Scope realistis untuk solo/small team open source.

## MVP (v0.1.0) — **Done**

**Goal:** Satu container image production-usable untuk single Go binary dengan Process Supervisor + deploy UX via `launcher.sh`.

| Item | Status | Notes |
|------|--------|-------|
| Launcher sebagai PID 1 | **Done** | `cmd/launcher` |
| **Process Supervisor** | **Done** | [11-process-supervisor.md](./11-process-supervisor.md) |
| Restart policy (`never`, `on-failure`, `always`) | **Done** | ENV-driven |
| Crash loop protection | **Done** | `APP_RESTART_BURST` + window |
| Signal forwarding | **Done** | SIGTERM, SIGINT, SIGQUIT |
| Graceful shutdown | **Done** | `APP_SHUTDOWN_TIMEOUT` |
| Supervisor lifecycle logging | **Done** | stderr structured events |
| Health check (optional) | **Done** | HTTP probe, ENV-gated |
| Capture stdout/stderr | **Done** | Daily log files |
| Log retention delete | **Done** | On startup + daily |
| Log viewer HTTP | **Done** | List, view, download |
| Basic Auth viewer | **Done** | ENV credentials |
| ENV + CLI configuration | **Done** | Full set in config doc |
| Dockerfile runtime image | **Done** | Alpine, non-root |
| **`launcher.sh` deploy wrapper** | **Done** | Multi-app, wizard, interactive menu |
| **`apps/` folder layout** | **Done** | Demo `http-server`, `hello` |
| README + design docs | **Done** | `docs/` |

**Out of MVP (runtime):**

- Unified log dashboard (satu port, pilih folder) — V3
- Metrics export, search, tail, gzip logs, plugin system

**Multi-app today:** beberapa container (satu app per container) via `launcher.sh`, bukan multi-process dalam satu container.

**Definition of Done:** verified — lihat [CHANGELOG.md](../CHANGELOG.md) dan [12-development-tasks.md](./12-development-tasks.md).

---

## V2 (v0.2.x – v0.3.x)

**Goal:** Operability improvements for production teams.

| Item | Priority | Description |
|------|----------|-------------|
| Realtime tail | High | SSE or WebSocket `/logs/{file}/tail` |
| Search | High | Query param search across file or dir |
| Pagination | Medium | Large log file offset/limit |
| Highlight ERROR | Medium | HTML view regex highlight |
| Gzip log archive | Medium | `LOG_GZIP_AFTER_DAYS` |
| Metrics | Medium | Prometheus `/metrics` on `METRICS_PORT` |
| TCP health check | Medium | `HEALTHCHECK_TYPE=tcp` |
| IP allowlist | Low | `VIEWER_ALLOW_CIDRS` |
| Distroless image variant | Low | Smaller attack surface |
| CI release + multi-arch | High | GitHub Actions, ghcr.io |

---

## V3 (v1.0.0+)

**Goal:** Platform features — unified ops UI and extensibility.

| Item | Description |
|------|-------------|
| Unified dashboard | Satu portal: pilih app/folder log, status multi-app |
| Multi-process single container | N binaries dari manifest (bukan hanya N container) |
| Auto update | Pull new launcher/binary (opt-in) |
| Plugin system | Hooks: pre-start, post-stop, log filter |
| Privilege drop | `APP_USER` / `APP_GROUP` |
| Config file | YAML/TOML alternative to ENV-only |
| `APP_ARGS` shell quoting | Robust arg parsing |

**Note:** Multi-app via `launcher.sh` (N container) sudah ada di v0.1.0; V3 fokus unified UI dan single-container multi-process.

---

## Milestone Timeline (Indicative)

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| MVP | Done | v0.1.0 — supervisor + logging + viewer + launcher.sh |
| V2 | 6–10 weeks | tail, search, metrics |
| V3 | TBD | unified dashboard RFC |

---

## Open Questions

| Question | Target resolution |
|----------|-------------------|
| Embed static UI vs server templates? | MVP: templates |
| Duplicate child output to `docker logs`? | MVP: no; revisit V2 |
| Health check kill vs restart only? | MVP: stop + restart per policy |
| Cosign image signing? | V2 |
| Unified log viewer vs per-app port? | V3 dashboard |

---

## Contribution Path

1. V2: tail protocol, search, metrics.
2. Good first issue: viewer pagination, gzip rotator.
3. V3 RFC: unified dashboard.

---

## Referensi

- Supervisor: [11-process-supervisor.md](./11-process-supervisor.md)
- Deploy: [08-docker.md](./08-docker.md)
- Architecture: [02-architecture.md](./02-architecture.md)
- Config: [04-configuration.md](./04-configuration.md)
- Tasks: [12-development-tasks.md](./12-development-tasks.md)
