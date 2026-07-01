# Development Tasks

Dokumen tracker implementasi QoLauncher. Gunakan untuk GitHub Issues, Projects, atau kanban lokal.

**Status legend:** `todo` · `in_progress` · `done` · `blocked` · `cancelled`

**Priority:** `P0` critical path · `P1` MVP required · `P2` MVP nice-to-have · `P3` post-MVP

**Doc refs:** nomor file di `docs/` (mis. `04` = `04-configuration.md`).

---

## Ringkasan Progress

| Milestone | Total | Done | In Progress | Todo |
|-----------|-------|------|-------------|------|
| MVP (v0.1.0) | 51 | 51 | 0 | 0 |
| Deploy UX (`launcher.sh`) | 7 | 7 | 0 | 0 |
| V2 | 12 | 0 | 0 | 12 |
| V3 | 8 | 0 | 0 | 8 |

> Update tabel ini saat task selesai.

---

## MVP v0.1.0 — Epic Overview

| Epic | ID Prefix | Tasks | Doc |
|------|-----------|-------|-----|
| Project scaffold | `SC` | 4 | 02 |
| Config & CLI | `CFG` | 6 | 04, 09 |
| Launcher logger | `LOG` | 2 | 05, 11 |
| Log writer & capture | `CAP` | 5 | 05 |
| Log rotator | `ROT` | 2 | 05 |
| Process supervisor | `SUP` | 12 | 11, 03 |
| Log viewer | `VIEW` | 7 | 06, 07 |
| Main & shutdown | `MAIN` | 3 | 02, 03 |
| Docker & deploy | `DOC` | 5 | 08 |
| Deploy wrapper | `WRAP` | 6 | 08, apps/ |
| Testing & QA | `QA` | 4 | 10 |
| Docs & release | `REL` | 2 | 01, 10 |

---

## Epic SC — Project Scaffold

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| SC-01 | Init Go module `github.com/raqolbi/qolauncher` ([repo](https://github.com/raqolbi/QoLauncher)) | P0 | done | — | `go.mod`, `.gitignore`, license (MIT) |
| SC-02 | Struktur direktori package | P0 | done | SC-01 | `cmd/launcher`, `internal/config`, `internal/supervisor`, `internal/capture`, `internal/logwriter`, `internal/rotator`, `internal/viewer`, `internal/logger` |
| SC-03 | Makefile / task runner dasar | P2 | done | SC-01 | Target: `build`, `test`, `lint`, `docker-build` |
| SC-04 | CI skeleton (GitHub Actions) | P2 | done | SC-01 | Workflow: `go test ./...` on push |

---

## Epic CFG — Config & CLI

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| CFG-01 | Struct `Config` + defaults | P0 | done | SC-02 | Semua field ENV di `04` terdefinisi |
| CFG-02 | Parser ENV | P0 | done | CFG-01 | Load dari `os.Environ()`, map ke struct |
| CFG-03 | Parser CLI flags | P0 | done | CFG-01 | Semua flag di `09`; CLI > ENV > default |
| CFG-04 | Validasi config fail-fast | P0 | done | CFG-02, CFG-03 | Error messages sesuai `04`; exit `1` pre-start |
| CFG-05 | `--config` print resolved (redact password) | P1 | done | CFG-04 | Password `***REDACTED***`; no side effects |
| CFG-06 | `--version` / `--help` | P1 | done | SC-02 | Exit `0`; format version di `09` |

**CFG-04 checklist validasi:**

- [x] `APP_BINARY` required & exists
- [x] `APP_RESTART_POLICY` enum valid
- [x] `APP_PORT` 0 atau 1–65535
- [x] `LOG_DIR` writable
- [x] Auth required if `VIEWER_ENABLED=true`
- [x] `HEALTHCHECK_URL` required if `HEALTHCHECK_ENABLED=true`
- [x] `APP_RESTART_BURST >= 1`
- [x] `LOG_PORT != APP_PORT` (jika keduanya > 0)

---

## Epic LOG — Launcher Logger

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| LOG-01 | Structured logger ke stderr | P0 | done | SC-02 | Level `debug/info/warn/error`; field key=value |
| LOG-02 | Supervisor event helpers | P0 | done | LOG-01 | Semua event di `11` (started, crashed, restarting, dll.) |

---

## Epic CAP — Log Writer & Capture

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| CAP-01 | Daily log writer `YYYY-MM-DD.log` | P0 | done | SC-02 | Append mode; TZ-aware rollover |
| CAP-02 | Line format `{ts} [stdout\|stderr] {msg}` | P0 | done | CAP-01 | RFC3339 offset; sesuai `05` |
| CAP-03 | Stdout pipe capture goroutine | P0 | done | CAP-02 | Line-buffered; partial line flush on EOF |
| CAP-04 | Stderr pipe capture goroutine | P0 | done | CAP-02 | Stream tag `[stderr]` |
| CAP-05 | Mutex / serialized writes | P0 | done | CAP-03, CAP-04 | No interleaved broken lines |

---

## Epic ROT — Log Rotator

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| ROT-01 | Retention sweep on startup | P1 | done | CAP-01 | Delete `*.log` older than `LOG_RETENTION_DAYS` |
| ROT-02 | Periodic sweep (24h ticker) | P2 | done | ROT-01 | `LOG_RETENTION_DAYS=0` → skip delete |

---

## Epic SUP — Process Supervisor

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| SUP-01 | `exec.Cmd` start child | P0 | done | CFG-04, CAP-03 | `APP_BINARY`, `APP_ARGS`, `APP_WORKDIR`, inherit ENV |
| SUP-02 | Wait exit + capture exit code | P0 | done | SUP-01 | Distinguish exit 0 vs non-zero vs signaled |
| SUP-03 | Restart policy `never` | P0 | done | SUP-02 | Child exit → launcher exit same code |
| SUP-04 | Restart policy `on-failure` | P0 | done | SUP-03 | Restart if exit ≠ 0; exit 0 → stop |
| SUP-05 | Restart policy `always` | P0 | done | SUP-04 | Restart on any exit except intentional shutdown |
| SUP-06 | `APP_RESTART_DELAY` between restarts | P0 | done | SUP-04 | Parse duration; sleep before re-exec |
| SUP-07 | Crash loop guard (burst + window) | P0 | done | SUP-06 | 10 restarts / 60s default → launcher exit `1` |
| SUP-08 | `APP_MAX_RESTART` lifetime cap | P1 | done | SUP-06 | `0` = unlimited; N exceeded → exit `1` |
| SUP-09 | Signal forwarding SIGTERM/SIGINT/SIGQUIT | P0 | done | SUP-01 | Forward to child; set intentional shutdown flag |
| SUP-10 | Graceful shutdown + `APP_SHUTDOWN_TIMEOUT` | P0 | done | SUP-09 | Wait → SIGKILL; log forced/graceful |
| SUP-11 | Health check HTTP probe (optional) | P1 | done | SUP-01 | ENV-gated; N failures → stop + restart per policy |
| SUP-12 | Filter `LAUNCHER_*` dari child ENV | P2 | done | SUP-01 | Reserved vars not passed to child |

---

## Epic VIEW — Log Viewer

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| VIEW-01 | HTTP server on `LOG_PORT` | P0 | done | CFG-04 | Bind `0.0.0.0`; respect `VIEWER_ENABLED` |
| VIEW-02 | Basic Auth middleware | P0 | done | VIEW-01 | Constant-time compare; 401 + WWW-Authenticate |
| VIEW-03 | `GET /` → redirect `/logs` | P2 | done | VIEW-01 | 302 |
| VIEW-04 | `GET /logs` JSON file list | P0 | done | VIEW-02 | Sorted newest first; name, date, size, modified |
| VIEW-05 | `GET /logs/{filename}` view | P0 | done | VIEW-04 | Path traversal blocked; `format=html\|raw` |
| VIEW-06 | `GET /logs/{filename}/download` | P0 | done | VIEW-04 | Content-Disposition attachment |
| VIEW-07 | `GET /health` viewer health | P1 | done | VIEW-01 | JSON `{status, viewer}`; no auth MVP |

---

## Epic MAIN — Main Entry & Orchestration

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| MAIN-01 | `cmd/launcher/main.go` init sequence | P0 | done | CFG, LOG, ROT, VIEW, SUP | Urutan startup sesuai `03` |
| MAIN-02 | Shutdown orchestration | P0 | done | MAIN-01, SUP-10 | Stop viewer → flush logs → exit code benar |
| MAIN-03 | Exit code propagation matrix | P0 | done | SUP-03–07 | Semua skenario di `11` terimplementasi |

---

## Epic DOC — Docker & Deploy

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| DOC-01 | Dockerfile multi-stage | P0 | done | SC-01 | Static binary; non-root user; `tzdata` |
| DOC-02 | `docker-compose.yml` example | P0 | done | DOC-01 | Restart policy, volumes, grace period |
| DOC-03 | Example app `apps/hello` | P1 | done | SC-01 | Minimal `fmt.Println` binary |
| DOC-04 | Example app `apps/http-server` | P1 | done | SC-01 | `APP_PORT` env + `/health` |
| DOC-05 | `.env.example` | P1 | done | CFG-01 | Semua ENV MVP documented |

---

## Epic WRAP — `launcher.sh` (Deploy UX)

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| WRAP-01 | `launcher.sh` interactive menu | P1 | done | DOC-01 | Run/Stop/Restart/Status/Logs/Apps/Setup |
| WRAP-02 | First-run wizard (`.env` + compose) | P1 | done | WRAP-01 | Wizard buat config global |
| WRAP-03 | Auto-detect apps di `apps/` | P1 | done | WRAP-01 | Subdir + flat mode; skip `*.go` |
| WRAP-04 | Multi-app compose generation | P1 | done | WRAP-03 | Satu service per app; port validation |
| WRAP-05 | Pilih app: all / satu / beberapa | P1 | done | WRAP-04 | Run: semua di apps/; Stop/Restart/Logs: hanya running |
| WRAP-06 | `apps/README.md` + Makefile targets | P2 | done | WRAP-01 | `make compose-up`, `build-examples` → `apps/` |
| WRAP-07 | Banner `Running:` live + menu no-exit on error | P2 | done | WRAP-01 | UX: docker ps, `|| true` on menu case |

---

## Epic QA — Testing & QA

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| QA-01 | Unit tests `config` validation | P0 | done | CFG-04 | Table-driven; all error cases |
| QA-02 | Unit tests rotator date logic | P1 | done | ROT-01 | Retention edge cases |
| QA-03 | Integration test supervisor restart | P0 | done | SUP-04, SUP-05 | Fake crashing binary; assert restart count |
| QA-04 | Integration test crash loop guard | P0 | done | SUP-07 | Fast-crash binary → launcher exit `1` |
| QA-05 | Manual QA checklist (DoD) | P0 | done | MAIN-01 | Verified via tests + CHANGELOG DoD table |

---

## Epic REL — Docs & Release

| ID | Task | Priority | Status | Deps | Acceptance Criteria |
|----|------|----------|--------|------|---------------------|
| REL-01 | `README.md` quickstart | P0 | done | DOC-02 | Link ke `docs/`; minimal run example |
| REL-02 | Tag `v0.1.0` + release notes | P0 | done | QA-05 | CHANGELOG.md; tag instructions in CHANGELOG |

---

## Definition of Done — MVP v0.1.0

Checklist akhir sebelum release:

- [x] **DoD-01** Deploy `apps/http-server` via `launcher.sh` / compose, `APP_RESTART_POLICY=on-failure`
- [x] **DoD-02** Kill app process → auto restart; log restart event di stderr launcher
- [x] **DoD-03** App stdout/stderr muncul di log harian + viewer
- [x] **DoD-04** `docker stop` → graceful shutdown dalam `APP_SHUTDOWN_TIMEOUT`
- [x] **DoD-05** Crash loop: 10+ crash dalam 60s → launcher exit `1`, container stop
- [x] **DoD-06** Log viewer: login Basic Auth, list, view, download
- [x] **DoD-07** Policy `never`: child exit code = container exit code
- [x] **DoD-08** `apps/hello` jalan dengan `APP_PORT=0`, viewer di port terpisah
- [x] **DoD-09** `launcher --config` dan `--version` berfungsi
- [x] **DoD-10** Semua unit test pass; no P0 task `todo`

---

## Urutan Implementasi (Suggested)

```
Phase 1 — Foundation
  SC-01 → SC-02 → CFG-01 → CFG-02 → CFG-04 → LOG-01

Phase 2 — Logging
  CAP-01 → CAP-02 → CAP-03 → CAP-04 → CAP-05 → ROT-01

Phase 3 — Supervisor core
  SUP-01 → SUP-02 → SUP-09 → SUP-10 → SUP-03 → SUP-04 → SUP-05
  → SUP-06 → SUP-07 → LOG-02

Phase 4 — Viewer
  VIEW-01 → VIEW-02 → VIEW-04 → VIEW-05 → VIEW-06 → VIEW-07

Phase 5 — Integration
  MAIN-01 → MAIN-02 → MAIN-03 → CFG-03 → CFG-05 → CFG-06
  → SUP-08 → SUP-11 → ROT-02

Phase 6 — Ship
  DOC-* → QA-* → REL-*
```

---

## Backlog V2

| ID | Task | Priority | Status | Doc |
|----|------|----------|--------|-----|
| V2-01 | Realtime tail SSE/WebSocket | P1 | todo | 06 |
| V2-02 | Log search endpoint | P1 | todo | 06 |
| V2-03 | Pagination large files | P2 | todo | 06 |
| V2-04 | ERROR highlight HTML view | P2 | todo | 06 |
| V2-05 | Gzip archive `LOG_GZIP_AFTER_DAYS` | P2 | todo | 04 |
| V2-06 | Prometheus metrics `METRICS_PORT` | P2 | todo | 04 |
| V2-07 | TCP health check type | P2 | todo | 11 |
| V2-08 | IP allowlist `VIEWER_ALLOW_CIDRS` | P3 | todo | 07 |
| V2-09 | Distroless Docker image | P3 | todo | 08 |
| V2-10 | CI release pipeline + ghcr.io | P1 | todo | 08 |
| V2-11 | Multi-arch buildx | P1 | todo | 08 |
| V2-12 | Cosign image signing | P3 | todo | 10 |

---

## Backlog V3

| ID | Task | Priority | Status | Doc |
|----|------|----------|--------|-----|
| V3-01 | Unified multi-app dashboard | P1 | todo | 10 |
| V3-02 | Multi-process single container | P2 | todo | 10 |
| V3-03 | Auto update (opt-in) | P3 | todo | 10 |
| V3-04 | Plugin hooks system | P2 | todo | 10 |
| V3-05 | Privilege drop `APP_USER` | P2 | todo | 10 |
| V3-06 | YAML/TOML config file | P2 | todo | 10 |
| V3-07 | Shell-style `APP_ARGS` quoting | P2 | todo | 04 |
| V3-08 | Multi-app RFC document | P0 | todo | 10 |

---

## GitHub Issues Template

Salin saat buat issue:

```markdown
## Task
ID: SUP-07
Title: Crash loop guard (burst + window)

## Description
Implement sliding window restart counter per docs/11-process-supervisor.md.

## Acceptance Criteria
- [ ] APP_RESTART_BURST default 10
- [ ] APP_RESTART_WINDOW default 60s
- [ ] Exceeded → log + launcher exit 1
- [ ] Log event `crash loop detected`

## Doc Ref
- docs/11-process-supervisor.md
- docs/04-configuration.md

## Dependencies
- SUP-06
```

---

## Changelog Tracker

| Date | Task ID | Author | Note |
|------|---------|--------|------|
| 2026-07-01 | SC-01–SC-04 | — | Epic SC selesai: module, packages, Makefile, CI |
| 2026-07-01 | CFG-01–CFG-06 | — | Epic CFG selesai: config, ENV/CLI, validate, --config/--help/--version |
| 2026-07-01 | LOG/CAP/ROT | — | Logger, logwriter, capture, rotator + init wiring di main |
| 2026-07-01 | SUP-01–SUP-12 | — | Process supervisor, restart policy, signals, health check |
| 2026-07-01 | VIEW-01–VIEW-07, MAIN | — | Log viewer HTTP + wiring init/shutdown |
| 2026-07-01 | DOC-01–DOC-05 | — | Dockerfile, compose, apps/ examples, .env.example |
| 2026-07-01 | REL-01–REL-02, QA-05 | — | README, CHANGELOG v0.1.0, DoD verified |
| 2026-07-01 | WRAP-01–WRAP-07 | — | launcher.sh, apps/, UX fixes |

---

## Referensi

| Doc | Isi |
|-----|-----|
| [02-architecture.md](./02-architecture.md) | Modul & lifecycle |
| [04-configuration.md](./04-configuration.md) | ENV reference |
| [08-docker.md](./08-docker.md) | Docker + `launcher.sh` |
| [10-roadmap.md](./10-roadmap.md) | Milestone |
| [11-process-supervisor.md](./11-process-supervisor.md) | Supervisor spec |
