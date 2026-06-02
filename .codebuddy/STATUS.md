# LightBoot Development Status

## Stage 0 — Project Skeleton & Toolchain ✅ COMPLETE
- Go module, directory scaffold, config system, CLI, Makefile

## Stage 1 — Database, Models & Event Bus ✅ COMPLETE
- SQLite via modernc.org/sqlite, migration, ISO model, event bus, scanner skeleton, SSE logs

## Stage 2 — Profile System & ISO Detection ✅ COMPLETE
- YAML profile loader, ISO reader (iso9660 + xorriso fallback), detector engine, 6 built-in profiles

## Stage 3 — ISO Extraction, Caching & Menu Generation ✅ COMPLETE
- Kernel/initrd extraction, cache manager, iPXE menu generator, HTTP boot file server

## Stage 4 — DHCP Proxy & TFTP Services ✅ COMPLETE
- ProxyDHCP on UDP 67, TFTP server on UDP 69, architecture detection via DHCP option 93

## Stage 5 — Web UI (Vue.js SPA) ✅ COMPLETE (source ready, needs npm build)
- Vue 3 + Vite + Vue Router + MDI icons scaffold under web/src/
- 6 views: Login, Dashboard, ISO Library, Boot Menu Preview, Logs, System Status
- API client with Bearer token auth, EventSource SSE, XHR upload with progress
- Go backend: auth middleware, CORS-ready, all /api/isos CRUD endpoints, upload, scan trigger, token regen
- SPA handler with index.html fallback for hash router deep links
- Makefile updated with `make web` (npm install + vite build)
- Placeholder dist page with build instructions when Vue not yet compiled

## Stage 6 — Documentation & Help System ✅ COMPLETE

### 6.1 MkDocs Setup
- MkDocs + Material theme installed in .mkdocs-venv
- 11 documentation files: index.md, installation.md, configuration.md, iso-management.md, profiles.md, boot-modes.md, networking.md, api-reference.md, secure-boot.md, troubleshooting.md, contributing.md
- mkdocs.yml: Material theme with slate palette, search plugin, navigation tabs, code copy features
- site_dir: site/ (output for embedding)
- Search index: 83KB search_index.json with full-text search
- 12 HTML pages generated, sitemap.xml present

### 6.2 Build Integration
- docs_embed.go: `//go:embed site` embeds built docs into Go binary
- `make docs` runs `.mkdocs-venv/bin/mkdocs build`
- router.go: serves embedded docs at `/docs/` route

### 6.3 Web UI Integration
- App.vue sidebar: "Help" link with `href="/docs/"` opens documentation in new tab

### 6.4 CLI Integration
- `lightboot help` prints: "LightBoot full documentation available at: http://localhost:8080/docs"

### Stage 6 File Inventory
| File | Purpose |
|------|--------|
| mkdocs.yml | MkDocs configuration with Material theme (MODIFIED — fixed broken icon) |
| docs_embed.go | Embeds site/ directory into Go binary (PRE-EXISTING) |
| internal/core/router.go | /docs/ route serving embedded docs (PRE-EXISTING) |
| cmd/lightboot/main.go | CLI help text with docs URL (PRE-EXISTING) |
| web/src/App.vue | Help link in sidebar (PRE-EXISTING) |
| site/ | 12 HTML files + search index + assets (REBUILT after icon fix) |

## Stage 7 — Security & Hardening ✅ COMPLETE
- CSP middleware: Content-Security-Policy, X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Referrer-Policy on all responses
- Rate limiter: per-IP auth failure tracking (10 attempts/min, 5min block) wired into AuthMiddleware
- ISO magic bytes verification: checks for CD001 at offset 32769 before accepting uploads
- SECURITY.md: vulnerability reporting policy with security design overview
- All security features compile and verified with make build + smoke test

### Stage 7 File Inventory
| File | Purpose |
|------|--------|
| internal/core/csp.go | CSP + security headers middleware (pre-existing, wired in this stage) |
| internal/core/ratelimit.go | Per-IP rate limiter for auth endpoints (NEW) |
| internal/core/auth.go | Added rate limiter parameter, RecordFailure on invalid token (MODIFIED) |
| internal/core/router.go | Magic bytes check on ISO upload, RateLimiter in deps (MODIFIED) |
| cmd/lightboot/main.go | CSP middleware wrapping, RateLimiter creation (MODIFIED) |
| SECURITY.md | Vulnerability reporting policy and security design overview (NEW) |

## Stage 8 — Docker & Distribution ✅ COMPLETE

### Stage 8 File Inventory
| File | Purpose |
|------|--------|
| docker/Dockerfile | Multi-stage build (golang + debian-slim runtime) with healthcheck |
| .dockerignore | Excludes node_modules, dist, git, config.yaml, IDE files |
| docker-compose.yml | One-command local deployment with volume mounts and NET_RAW cap |
| .github/workflows/ci.yml | CI/CD pipeline: vet, test, Docker push to GHCR, release artifacts |
| Makefile | Added docker-up, docker-down, docker-logs targets |

## Stage 9 — Polishing & Community Readiness ✅ COMPLETE

### 9.1 Additional Built-in Profiles
- VMware ESXi stub (`internal/core/profiles/vmware-esxi.yaml`) — manual setup instructions
- Windows 11 stub (`internal/core/profiles/windows-11.yaml`) — wimboot/WinPE manual setup
- Kali Linux, Pop_OS!, Linux Mint, Proxmox VE already in embedded profiles directory

### 9.2 Error Handling & UX Polish
- Dashboard.vue: loading spinner, error state, proper empty state for ISOs table
- ISOLibrary.vue: loading spinner, fetch error display, empty states for filters
- SystemStatus.vue: spinner while fetching config, error display, proper "no data" state
- MenuPreview.vue: spinner, error display, fallback text
- App.vue: `.spinner` CSS animation added globally

### 9.3 Logging Polish — File Output
- Logger struct: added `file *os.File` and `mu sync.Mutex` fields
- `SetLogFile(path)` method: opens/creates log file with append mode (0644)
- `Close()` method: safely closes log file on shutdown
- `log()` method: writes `[RFC3339] [level] [source] message` to file when configured
- main.go: wires `cfg.LogFile` to `logger.SetLogFile()` with error handling + deferred `logger.Close()`
- config.go: `LogFile` field + `LIGHTBOOT_LOG_FILE` env var override
- router.go: `log_file` exposed in `/api/config` endpoint

### Stage 9 File Inventory
| File | Purpose |
|------|--------|
| internal/core/profiles/vmware-esxi.yaml | VMware ESXi stub profile (NEW) |
| internal/core/profiles/windows-11.yaml | Windows 11 stub profile (NEW) |
| internal/core/logger.go | Added os import, file+mu fields, SetLogFile, Close, file write in log() (MODIFIED) |
| internal/core/config.go | LogFile field + LIGHTBOOT_LOG_FILE env override (MODIFIED) |
| internal/core/router.go | log_file exposed in /api/config response (MODIFIED) |
| cmd/lightboot/main.go | Wire SetLogFile + defer Close (MODIFIED) |
| web/src/App.vue | Spinner CSS animation (MODIFIED) |
| web/src/views/Dashboard.vue | Loading/error/empty states (MODIFIED) |
| web/src/views/ISOLibrary.vue | Loading/fetchError states (MODIFIED) |
| web/src/views/SystemStatus.vue | Loading/fetchError config display (MODIFIED) |
| web/src/views/MenuPreview.vue | Loading/error display (MODIFIED) |

---

## Stage 5 Detailed File Inventory

### Vue.js Source (web/src/)
| File | Purpose |
|------|--------|
| web/src/main.js | App entry, router setup, MDI import |
| web/src/api.js | fetch/XHR wrapper, token mgmt, SSE helper |
| web/src/App.vue | Root layout: sidebar nav + router-view |
| web/src/views/Login.vue | Token-based login page |
| web/src/views/Dashboard.vue | ISO stats, recent ISOs, rescan button |
| web/src/views/ISOLibrary.vue | ISO table, drag-drop upload, filter, delete |
| web/src/views/MenuPreview.vue | iPXE boot menu syntax preview |
| web/src/views/LogsView.vue | Live SSE log stream with filter |
| web/src/views/SystemStatus.vue | Service health, config table, token mgmt |

### Backend (internal/core/)
| File | Purpose |
|------|--------|
| internal/core/auth.go | Bearer token auth middleware (NEW) |
| internal/core/router.go | All HTTP routes: public boot + authed API + SPA fallback (REWRITTEN) |

### Project Files
| File | Purpose |
|------|--------|
| web/package.json | Vue dependencies (vue, vue-router, @mdi/font, vite) |
| web/vite.config.js | Vite config with dev proxy to :8080 |
| web/index.html | Vite entry HTML |
| web/dist/index.html | Placeholder page (shown when SPA not built) |
| Makefile | Updated `make web` target |

### To build the Web UI:
```
cd web && npm install && npx vite build
```
or:
```
make web
```
