# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bolt Card Hub (Phoenix Edition) — a lightweight Bitcoin/Lightning payment service for hosting NFC Bolt Cards. Integrates with Phoenix Server (phoenixd) for Lightning Network payments. Written in Go, deployed via Docker Compose.

## Build & Run

```bash
# First-time setup: create .env from example and set your domain
cp .env.example .env
# Edit .env to set HOST_DOMAIN=hub.yourdomain.com

# Build and run
docker compose build
docker compose up        # foreground
docker compose up -d     # detached

# Development with hot reload (rebuilds card container on source changes)
docker compose watch
```

The Go application is in `docker/card/`. It builds via multi-stage Docker build (see `docker/card/Dockerfile`). Build flags inject version/date/time into `card/build`.

There is no Makefile — building is done exclusively through Docker.

## Testing

```bash
# Run all tests (from docker/card/)
cd docker/card && go test -race -count=1 ./...

# Run specific test packages
go test ./crypto/    # AES-CMAC and AES decrypt tests
go test ./db/        # Schema migration, settings CRUD, card operations (uses in-memory SQLite)
go test ./web/       # HTTP handler tests (auth, balance, path traversal)
```

Tests require CGo (for `go-sqlite3`) and `HOST_DOMAIN` env var (db_init panics without it; test helpers set it automatically). CI runs tests automatically via GitHub Actions on push/PR to main.

## CI

GitHub Actions workflow (`.github/workflows/ci.yml`) runs on push/PR to `main`:
- `go vet ./...`
- `go build`
- `go test -race -count=1 ./...`
- `govulncheck ./...`
- Docker image builds for both `card` and `webproxy`

Uses Go 1.25.7 with CGo enabled for sqlite3.

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
- **card**: Custom Go app — card service on `:8000` (192M memory, GOMEMLIMIT=150MiB). Has Docker healthcheck (HEAD / every 30s). Graceful shutdown on SIGTERM with 10s drain timeout. Includes `sqlite3` for database access.
- **webproxy**: Custom Caddy build (via xcaddy with `caddy-ratelimit` plugin) — reverse proxy with auto-TLS, CORS, zstd compression, and rate limiting on auth endpoints (10 req/min per IP on `/admin/login/`, `/auth`, `/pos/auth`)

All on internal `hubnet` bridge network. Card container mounts phoenix volume read-only for config access. `HOST_DOMAIN` is set in `.env` and shared with both card and webproxy containers via `env_file`. The Caddyfile uses `{$HOST_DOMAIN}` for the site address — no templating or init scripts needed.

### Go Application (`docker/card/`)

Entry point: `main.go` → opens SQLite DB → runs CLI or starts HTTP server on `:8000`

**Packages:**
- `web/` — HTTP handlers using Gorilla Mux. Handler pattern: `func (app *App) CreateHandler_Name() http.HandlerFunc`
- `db/` — SQLite operations split by verb: `db_select.go`, `db_get.go`, `db_set.go`, `db_insert.go`, `db_update.go`, `db_add.go`, `db_wipe.go`. Schema init and migrations in `db_init.go`/`db_create.go`
- `phoenix/` — HTTP client for Phoenix Server API (invoices, payments, balance, channels). Uses basic auth from phoenix config
- `crypto/` — AES-CMAC authentication and AES decryption for Bolt Card NFC protocol
- `util/` — Error handling helpers (`CheckAndLog`), random hex generation, QR code encoding. Note: `CheckAndPanic` exists but must not be used in HTTP handlers — use inline error handling instead.
- `build/` — Version string (currently "0.9.2"), date/time injected at build
- `web-content/` — HTML templates (loaded into memory at startup) and static assets under `public/`

### Route Groups (`web/app.go`)

- `/ln`, `/cb` — LNURL-withdraw protocol (NFC card tap → payment)
- `/admin/` — Admin dashboard (cookie-based session auth)
- `/new`, `/wipe` — Bolt Card Programmer endpoints
- BoltCardHub API (`/create`, `/auth`, `/balance`, `/payinvoice`, etc.) — LndHub-compatible, feature-gated via `bolt_card_hub_api` setting
- PoS API (`/pos/`) — Point-of-Sale subset of LndHub API, feature-gated via `bolt_card_pos_api` setting
- `/websocket` — Real-time payment notifications

### Database

SQLite at `/card_data/cards.db` with WAL mode, FULL synchronous, foreign keys, secure delete.

**Tables:** `settings` (key-value config), `cards` (card keys/auth/limits), `card_payments` (spending), `card_receipts` (loading/receiving), `program_cards` (batch programming)

Schema version managed by idempotent `update_schema_*` functions in `db_create.go`. Current schema version: 5.

### Authentication

- **Admin**: bcrypt password hash in settings table, session cookies. Legacy SHA256 hashes auto-migrate to bcrypt on login.
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

## Memory File

After completing a set of changes, update the persistent memory file at `~/.claude/projects/-home-user-boltcard-hub/memory/MEMORY.md` with any new patterns, conventions, or project facts discovered during the session. Keep it concise and organized by topic. This helps maintain context across conversations.