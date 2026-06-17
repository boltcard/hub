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

The Go application is in `docker/card/`. The admin UI is a React SPA in `docker/card/admin-ui/`. Both are built via a 3-stage Docker build: Node 22 (frontend) → Go 1.25.11 (backend) → Debian slim runtime (see `docker/card/Dockerfile`). Build flags inject version/date/time into `card/build`. Pre-built images are published to Docker Hub (`boltcard/card:latest`, `boltcard/webproxy:latest`).

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
- Docker image builds for both `card` and `webproxy` (via `docker/build-push-action@v6`)
- On push to `main` (not PRs): pushes images to Docker Hub as `latest`

Uses Go 1.25.11 with CGo enabled for sqlite3, Node 22 for frontend. Docker Hub push requires GitHub secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`.

> **Keep `provenance: false` on the image build steps.** `docker/build-push-action` enables provenance attestations by default, which wraps the pushed image in an OCI image index (no top-level `config`). The version check in deployed hubs `<=0.22.3` only parses a single-platform manifest, so an index leaves their "Update available" button permanently stuck (issue #45). Newer hubs handle the index too (`web/update.go` follows it), but the published `latest` must stay a single manifest so already-deployed hubs can ever see the update that fixes them.

## Versioning

**Bump the version for every PR.** Bump `Version` in `docker/card/build/build.go` as part of each PR, following [semantic versioning](https://semver.org/): a new backward-compatible feature bumps MINOR (e.g. `0.19.9` → `0.20.0`), a bug fix or maintenance change bumps PATCH (e.g. `0.19.3` → `0.19.4`), and a breaking change bumps MAJOR. On merge to `main`, CI republishes `boltcard/card:latest` with `org.opencontainers.image.version` set to this string; the About-page update check only surfaces the "Update available" button on deployed hubs when that label exceeds their running version. A PR that changes the image without bumping the version therefore ships an update no one can see.

## CLI Commands (run inside card container)

```bash
docker exec -it card bash
./app SendLightningPayment <invoice> <amountSat>
./app SetupCardAmountForTag <group_tag> <amount_sats>
./app ClearCardBalancesForTag <group_tag>
./app ProgramBatch <group_tag> <max_group_num> <initial_balance> <expiry_hours>
./app WipeCard <card_id>
./app DisableAdmin2FA
```

## Architecture

### Docker Services (docker-compose.yml)

- **phoenix**: acinq/phoenixd:0.8.0 — Lightning node (384M memory)
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
- `build/` — Version string (currently "0.22.4"), date/time injected at build
- `web-content/` — Static assets under `public/`, SPA build output under `admin/spa/`

### Route Groups (`web/app.go`)

- `/ln`, `/cb` — LNURL-withdraw protocol (NFC card tap → payment)
- `/admin` — React SPA admin UI (static assets + SPA index fallback, PathPrefix without trailing slash)
- `/admin/api/` — Admin JSON API (cookie-based session auth, 21 endpoints)
- `/new` — Bolt Card Programmer endpoint
- BoltCardHub API (`/create`, `/auth`, `/balance`, `/payinvoice`, etc.) — LndHub-compatible, feature-gated via `bolt_card_hub_api` setting
- PoS API (`/pos/`) — Point-of-Sale subset of LndHub API, feature-gated via `bolt_card_pos_api` setting
- `/admin/api/websocket` — Real-time payment notifications (JSON events via `wsHub` broadcast, requires admin session cookie)
- `/admin/api/phoenix/transactions` — Last 5 incoming/outgoing Phoenix payments
- `/admin/api/database/stats` — Database file size, schema version, table row counts
- `/admin/api/about/logs` — Last 20 container log lines (via Docker API, ANSI→HTML color conversion)
- `/admin/api/about/commits` — Last 10 GitHub commits (from GitHub API)
- `/admin/api/withdraw` — Admin fund withdrawal. `GET` returns node balance, total card liability, and spare/excess liquidity plus recent withdrawals; `POST` pays out node liquidity to a Lightning address. The POST handler re-verifies the admin password (on top of the session cookie), caps the amount at the node balance, and flags (`breachesLiability`) when the payout dips below outstanding card balances. See `web/admin_api_withdraw.go` and `phoenix/pay_lightning_address.go` (phoenixd `/paylnaddress`).

### Admin Update (`web/update.go`)

The About page (`/admin/about/`) checks for new versions by querying the Docker Hub registry API for the `org.opencontainers.image.version` label on the `boltcard/card:latest` image. This ensures the update button only appears when a newer image is actually available to pull (unlike the previous GitHub-based check which could race ahead of CI). The Dockerfile sets this label via `ARG APP_VERSION` (CI passes the real version, local builds get "unknown"). When an update is available, an "Update" button appears. Clicking it triggers the update mechanism:

1. Card container inspects itself via Docker API to find the compose project directory
2. Pulls `docker:cli` image and creates a disposable `hub-updater` container
3. The updater runs `docker compose pull && docker compose up -d` with AutoRemove
4. This avoids the self-update problem — the card container delegates to an independent container

Docker socket (`/var/run/docker.sock`) is mounted into the card container. The update endpoint is admin-only (behind session auth). All Docker API calls use Go stdlib `net/http` with Unix socket transport — no external dependencies. The frontend uses `onSettled` to always show an "Updating..." spinner and poll for the server to come back, avoiding a 502 error page during container restart.

### Database

SQLite at `/card_data/cards.db` with WAL mode, FULL synchronous, foreign keys, secure delete.

**Tables:** `settings` (key-value config), `cards` (card keys/auth/limits), `card_payments` (spending), `card_receipts` (loading/receiving), `program_cards` (batch programming), `pay_link_addresses` (rotating pay-link addresses), `admin_withdrawals` (admin payout audit log)

Schema version managed by idempotent `update_schema_*` functions in `db_create.go`. Current schema version: 12.

**Admin withdrawals:** the `admin_withdrawals` table (schema v12) is an audit log of admin-initiated payouts of node liquidity (paying out the hub's own funds, not tied to any card). Each row records the destination Lightning address, amount, routing fee, payment hash, and status (`pending`/`paid`/`failed`). See `db/db_admin_withdrawal.go`.

### Authentication

- **Admin**: bcrypt password hash in settings table, session cookies with 24-hour expiry. Legacy SHA256 hashes auto-migrate to bcrypt on login. Constant-time token comparison. Optional TOTP 2FA (RFC 6238 via `github.com/pquerna/otp`): when `admin_totp_enabled="Y"`, login also requires a 6-digit TOTP code or a single-use recovery code, verified in `adminApiLogin` before a session is issued (login-only enforcement). Enrollment/disable endpoints live in `web/admin_api_2fa.go`, TOTP/recovery helpers in `web/totp.go`. Disabling 2FA via the admin UI requires the password **and** a current TOTP/recovery code (proof of possession). Recovery for a lost authenticator: backup codes or the `DisableAdmin2FA` CLI command.
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
- `admin_totp_enabled`, `admin_totp_secret`, `admin_totp_recovery_hash` — optional admin login 2FA (TOTP). `admin_totp_enabled` ("Y"/"N") gates enforcement at login; `admin_totp_secret` (base32) and `admin_totp_recovery_hash` (JSON array of bcrypt-hashed single-use recovery codes) are redacted in the settings UI. Cleared by the `DisableAdmin2FA` CLI command.
- `new_card_code` — secret for card programming endpoint
- `invite_secret` — optional secret for wallet API card creation
- `bolt_card_hub_api`, `bolt_card_pos_api` — feature flags ("enabled" to activate)
- `lnurlw_k1_timeout_seconds` — optional validity window (seconds) for the LNURLw `k1` token in `lnurlw_request.go` (defaults to 10 if unset or invalid)
- `pay_link_expiry_days` — optional expiry (days) for generated LUD-19 pay links (defaults to 30 if unset or invalid)
- `schema_version_number` — tracks database migration state

Withdraw limits (`min_withdraw_sats=1`, `max_withdraw_sats=100000000`) are hardcoded in `lnurlw_request.go` and `lnurlw_callback.go`, not stored in the database.

> **Note (Phoenix routing fees):** the hub does *not* send a fee cap to phoenixd — `phoenix/send_lightning_payment.go` posts only `amountSat` + `invoice` to `/payinvoice`, so phoenixd applies its own default fee budget. The `max_network_fee_sats` value (`4 + amountSats*4/1000` in `lnurlw_callback.go`) is used only to reserve card balance, not as the payment fee ceiling. This bit us once: phoenixd 0.7.3 (lightning-kmp 1.11) couldn't route tiny ~20–250 sat payments within its default budget and rejected them with `"routing fees are insufficient"`; upgrading to 0.8.0 (lightning-kmp 1.12.0) fixed it. **Possible future hardening (may never be needed):** pass an explicit `maxFeeFlatSat` (sats) to `/payinvoice` — e.g. a `5 + amountSats*4/1000` floor — so the hub controls the fee budget instead of relying on phoenixd defaults.

The admin settings page (`/admin/settings/`) displays all settings with sensitive values (`_hash`, `_token`, `_code`, `_secret` suffixes) redacted. `log_level` has an inline dropdown that submits on change.

## Memory File

After completing a set of changes, update the persistent memory file at `~/.claude/projects/-home-debian-hub/memory/MEMORY.md` with any new patterns, conventions, or project facts discovered during the session. Keep it concise and organized by topic. This helps maintain context across conversations.