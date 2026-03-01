# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bolt Card Hub (Phoenix Edition) — a lightweight Bitcoin/Lightning payment service for hosting NFC Bolt Cards. Integrates with Phoenix Server (phoenixd) for Lightning Network payments. Written in Go, deployed via Docker Compose.

## Build & Run

```bash
# Production: pull pre-built images from Docker Hub and run
echo "HOST_DOMAIN=hub.yourdomain.com" > .env
docker compose pull
docker compose up -d

# Development: build locally and run
cp .env.example .env
# Edit .env to set HOST_DOMAIN=hub.yourdomain.com
docker compose build
docker compose up        # foreground
docker compose up -d     # detached

# Development with hot reload (rebuilds card container on source changes)
docker compose watch
```

The Go application is in `docker/card/`. The admin UI is a React SPA in `docker/card/admin-ui/`. Both are built via a 3-stage Docker build: Node 22 (frontend) → Go 1.25.7 (backend) → Debian slim runtime (see `docker/card/Dockerfile`). Build flags inject version/date/time into `card/build`. Pre-built images are published to Docker Hub (`boltcard/card:latest`, `boltcard/webproxy:latest`).

There is no Makefile — building is done exclusively through Docker.

## Testing

```bash
# Run all tests (from docker/card/)
cd docker/card && go test -race -count=1 ./...

# Run specific test packages
go test ./crypto/    # AES-CMAC and AES decrypt tests
go test ./db/        # Schema migration, settings CRUD, card operations (uses in-memory SQLite)
go test ./web/       # HTTP handler tests (auth, balance, path traversal, LNURL withdraw flow)
```

Tests require CGo (for `go-sqlite3`) and `HOST_DOMAIN` env var (db_init panics without it; test helpers set it automatically). CI runs tests automatically via GitHub Actions on push/PR to main.

## CI

GitHub Actions workflow (`.github/workflows/ci.yml`) runs on push/PR to `main`:
- `go vet ./...`
- `go build`
- `go test -race -count=1 ./...`
- `govulncheck ./...`
- Frontend build (`npm ci && npm run build` in `docker/card/admin-ui/`)
- Docker image builds for both `card` and `webproxy`
- On push to `main` (not PRs): pushes images to Docker Hub as `latest`

Uses Go 1.25.7 with CGo enabled for sqlite3, Node 22 for frontend. Docker Hub push requires GitHub secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`.

## CLI Commands (run inside card container)

```bash
docker exec -it card bash
./app SendLightningPayment <invoice> <amountSat>
./app SetupCardAmountForTag <group_tag> <amount_sats>
./app ClearCardBalancesForTag <group_tag>
./app ProgramBatch <group_tag> <max_group_num> <initial_balance> <expiry_hours>
./app WipeCard <card_id>
```

## Architecture

### Docker Services (docker-compose.yml)

- **phoenix**: acinq/phoenixd:0.7.2 — Lightning node (384M memory)
- **card**: Custom Go app — card service on `:8000` (192M memory, GOMEMLIMIT=150MiB). Has Docker healthcheck (HEAD / every 30s). Graceful shutdown on SIGTERM with 10s drain timeout. Includes `sqlite3` for database access. Mounts Docker socket for admin update feature.
- **webproxy**: Custom Caddy build (via xcaddy with `caddy-ratelimit` plugin) — reverse proxy with auto-TLS, CORS, zstd compression, and rate limiting on auth endpoints (10 req/min per IP on `/admin/login/`, `/auth`, `/admin/api/auth/login`)

All on internal `hubnet` bridge network. Card container mounts phoenix volume read-only for config access and Docker socket for self-update. `HOST_DOMAIN` is set in `.env` and shared with both card and webproxy containers via `env_file`. The Caddyfile uses `{$HOST_DOMAIN}` for the site address — no templating or init scripts needed.

### Go Application (`docker/card/`)

Entry point: `main.go` → opens SQLite DB → runs CLI or starts HTTP server on `:8000`

**Packages:**
- `admin-ui/` — React SPA (Vite + TypeScript + Tailwind v4 + shadcn/ui). Pages: login, dashboard, cards, card-detail, phoenix, settings, database, about
- `web/` — HTTP handlers using Gorilla Mux. Handler pattern: `func (app *App) CreateHandler_Name() http.HandlerFunc`. Admin API in `admin_api_*.go` files with `adminApiAuth()` middleware
- `db/` — SQLite operations split by verb: `db_select.go`, `db_get.go`, `db_set.go`, `db_insert.go`, `db_update.go`, `db_add.go`, `db_wipe.go`. Schema init and migrations in `db_init.go`/`db_create.go`
- `phoenix/` — HTTP client for Phoenix Server API (invoices, payments, balance, channels). Uses basic auth from phoenix config (password cached at startup with `sync.Once`)
- `crypto/` — AES-CMAC authentication and AES decryption for Bolt Card NFC protocol
- `util/` — Error handling helpers (`CheckAndLog`), random hex generation, QR code encoding
- `build/` — Version string (currently "0.16.0"), date/time injected at build
- `web-content/` — Static assets under `public/`, SPA build output under `admin/spa/`

### Route Groups (`web/app.go`)

- `/ln`, `/cb` — LNURL-withdraw protocol (NFC card tap → payment)
- `/admin/` — React SPA admin UI (static assets + SPA index fallback)
- `/admin/api/` — Admin JSON API (cookie-based session auth, 17 endpoints)
- `/new` — Bolt Card Programmer endpoint
- BoltCardHub API (`/create`, `/auth`, `/balance`, `/payinvoice`, etc.) — LndHub-compatible, feature-gated via `bolt_card_hub_api` setting
- PoS API (`/pos/`) — Point-of-Sale subset of LndHub API, feature-gated via `bolt_card_pos_api` setting
- `/websocket` — Real-time payment notifications (JSON events via `wsHub` broadcast)
- `/admin/api/phoenix/transactions` — Last 5 incoming/outgoing Phoenix payments
- `/admin/api/database/stats` — Database file size, schema version, table row counts

### Admin Update (`web/update.go`)

The About page (`/admin/about/`) checks for new versions by querying the Docker Hub registry API for the `org.opencontainers.image.version` label on the `boltcard/card:latest` image. This ensures the update button only appears when a newer image is actually available to pull (unlike the previous GitHub-based check which could race ahead of CI). The Dockerfile sets this label via `ARG APP_VERSION` (CI passes the real version, local builds get "unknown"). When an update is available, an "Update" button appears. Clicking it triggers the update mechanism:

1. Card container inspects itself via Docker API to find the compose project directory
2. Pulls `docker:cli` image and creates a disposable `hub-updater` container
3. The updater runs `docker compose pull && docker compose up -d` with AutoRemove
4. This avoids the self-update problem — the card container delegates to an independent container

Docker socket (`/var/run/docker.sock`) is mounted into the card container. The update endpoint is admin-only (behind session auth). All Docker API calls use Go stdlib `net/http` with Unix socket transport — no external dependencies. The frontend uses `onSettled` to always show an "Updating..." spinner and poll for the server to come back, avoiding a 502 error page during container restart.

### Database

SQLite at `/card_data/cards.db` with WAL mode, FULL synchronous, foreign keys, secure delete.

**Tables:** `settings` (key-value config), `cards` (card keys/auth/limits), `card_payments` (spending), `card_receipts` (loading/receiving), `program_cards` (batch programming)

Schema version managed by idempotent `update_schema_*` functions in `db_create.go`. Current schema version: 6.

### Authentication

- **Admin**: bcrypt password hash in settings table, session cookies with 24-hour expiry. Legacy SHA256 hashes auto-migrate to bcrypt on login. Constant-time token comparison.
- **Bolt Card NFC**: AES-CMAC with 5 keys per card (K0-K4), counter-based replay protection
- **Wallet/PoS API**: Login/password → access_token + refresh_token (random hex)

## Key Dependencies

- `gorilla/mux` — HTTP routing
- `mattn/go-sqlite3` — SQLite driver (CGo)
- `gorilla/websocket` — WebSocket
- `sirupsen/logrus` — Structured logging
- `nbd-wtf/ln-decodepay` — Lightning invoice decoding
- `aead/cmac` — AES-CMAC for Bolt Card auth
- `go-ini/ini` — Phoenix config file parsing
- `skip2/go-qrcode` — QR code generation
- `golang.org/x/crypto` — bcrypt password hashing

### Settings

The `settings` table stores key-value config. Active settings used by the app:
- `host_domain` — domain for building LNURL/callback URLs (set from `HOST_DOMAIN` env var on first run)
- `log_level` — logrus log level, applied at startup and changeable live via admin UI dropdown
- `admin_password_hash`, `admin_password_salt`, `admin_session_token`, `admin_session_created` — admin auth
- `new_card_code` — secret for card programming endpoint
- `invite_secret` — optional secret for wallet API card creation
- `bolt_card_hub_api`, `bolt_card_pos_api` — feature flags ("enabled" to activate)
- `schema_version_number` — tracks database migration state

Withdraw limits (`min_withdraw_sats=1`, `max_withdraw_sats=100000000`) are hardcoded in `lnurlw_request.go` and `lnurlw_callback.go`, not stored in the database.

The admin settings page (`/admin/settings/`) displays all settings with sensitive values (`_hash`, `_token`, `_code` suffixes) redacted. `log_level` has an inline dropdown that submits on change.

## Memory File

After completing a set of changes, update the persistent memory file at `~/.claude/projects/-home-debian-hub/memory/MEMORY.md` with any new patterns, conventions, or project facts discovered during the session. Keep it concise and organized by topic. This helps maintain context across conversations.