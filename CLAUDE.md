# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bolt Card Hub (Phoenix Edition) — a lightweight Bitcoin/Lightning payment service for hosting NFC Bolt Cards. Integrates with Phoenix Server (phoenixd) for Lightning Network payments. Written in Go, deployed via Docker Compose.

## Build & Run

```bash
# First-time setup (generates Caddyfile and Dockerfile from templates with domain name)
./docker_init.sh

# Build and run
docker compose build
docker compose up        # foreground
docker compose up -d     # detached

# Development with hot reload (rebuilds card container on source changes)
docker compose watch
```

The Go application is in `docker/card/`. It builds via multi-stage Docker build (see `docker/card/Dockerfile.template`). Build flags inject version/date/time into `card/build`.

There is no Makefile — building is done exclusively through Docker. There are no Go tests.

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

- **phoenix**: acinq/phoenixd:0.7.2 — Lightning node (256M memory)
- **card**: Custom Go app — card service on `:8000` (128M memory, GOMEMLIMIT=100MiB)
- **webproxy**: Caddy — reverse proxy with auto-TLS, CORS, zstd compression

All on internal `hubnet` bridge network. Card container mounts phoenix volume read-only for config access.

### Go Application (`docker/card/`)

Entry point: `main.go` → opens SQLite DB → runs CLI or starts HTTP server on `:8000`

**Packages:**
- `web/` — HTTP handlers using Gorilla Mux. Handler pattern: `func (app *App) CreateHandler_Name() http.HandlerFunc`
- `db/` — SQLite operations split by verb: `db_select.go`, `db_get.go`, `db_set.go`, `db_insert.go`, `db_update.go`, `db_add.go`, `db_wipe.go`. Schema init and migrations in `db_init.go`/`db_create.go`
- `phoenix/` — HTTP client for Phoenix Server API (invoices, payments, balance, channels). Uses basic auth from phoenix config
- `crypto/` — AES-CMAC authentication and AES decryption for Bolt Card NFC protocol
- `util/` — Error handling helpers (`CheckAndPanic`, `CheckAndLog`), random hex generation, QR code encoding
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

- **Admin**: Salted password hash in settings table, session cookies
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
