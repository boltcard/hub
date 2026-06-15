# Admin Login 2FA (TOTP) — Design

**Date:** 2026-06-15
**Status:** Approved (design)
**Target version:** 0.20.0

## Summary

Add **optional** TOTP-based two-factor authentication to the single-admin
login. When enabled, login requires a 6-digit TOTP code (or a one-time
recovery code) in addition to the existing admin password. The feature is:

- Implemented with `github.com/pquerna/otp` (no hand-rolled crypto).
- Stored in the existing `settings` key-value table — **no schema migration**.
- Enrolled via a QR-code flow in the admin UI Settings page.
- Recoverable via either one-time backup codes **or** a `docker exec` CLI
  command, so a lost authenticator never permanently locks the admin out.

Enforcement is **login-only**. The withdraw handler keeps its existing
password re-prompt; it does not additionally require a TOTP code.

## Decisions (locked during brainstorming)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Recovery | One-time recovery codes **and** a CLI disable command | Defense in depth; never weakens the second factor |
| TOTP impl | `pquerna/otp` library | Avoids subtle RFC 6238 / base32 / skew bugs |
| Enforcement | Login only | Session is already 2FA-backed; avoids 30s-code friction during a payout |
| Versioning | SemVer minor bump (0.19.8 → 0.20.0) | New backward-compatible feature |

### Non-goals (YAGNI)

- No TOTP requirement on the withdraw re-prompt (login-only enforcement).
- No separate backend rate-limiter — Caddy already rate-limits
  `/admin/api/auth/login` at 10 req/min/IP.
- No multi-admin / per-user 2FA — the system is single-admin by design.
- No "password still works as fallback" mode — that would defeat 2FA.

## Storage (settings table — no migration)

Three new keys in the `settings` table:

| Key | Value | Settings-UI visibility |
|-----|-------|------------------------|
| `admin_totp_enabled` | `"Y"` / `"N"` (or empty) | Shown — status only |
| `admin_totp_secret` | base32 TOTP shared secret | **Redacted** |
| `admin_totp_recovery_hash` | JSON array of bcrypt hashes of the **unused** recovery codes | **Redacted** (matches existing `_hash` suffix) |

**Redaction:** add a `_secret` suffix to the `strings.HasSuffix` guard in
`web/admin_api_settings.go` (currently `_hash` / `_token` / `_code`). The
recovery key already ends in `_hash`, so it is redacted with no change.

**Why the secret is plaintext but redacted:** TOTP is a shared-secret scheme,
so the secret must be stored recoverable to validate codes. Redaction keeps it
out of the settings list endpoint. Recovery codes, by contrast, are stored
**bcrypt-hashed** — a DB read never yields usable codes.

**Enforcement keys on `admin_totp_enabled == "Y"`**, never on the mere presence
of a secret. This guarantees a half-finished enrollment (secret generated but
code never confirmed) cannot lock anyone out.

## TOTP implementation

Package: `github.com/pquerna/otp` + `github.com/pquerna/otp/totp`.

- **Secret generation:** `totp.Generate(totp.GenerateOpts{Issuer: "Bolt Card
  Hub", AccountName: <host_domain>})` → returns an `*otp.Key` exposing
  `.Secret()` (base32) and `.URL()` (the `otpauth://` URI).
- **Validation:** `totp.ValidateCustom(code, secret, time.Now(),
  totp.ValidateOpts{Period: 30, Skew: 1, Digits: 6, Algorithm: SHA1})` —
  **±1 time-step skew** tolerates client/server clock drift.
- **QR rendering:** feed `key.URL()` into the **existing**
  `util.QrPngBase64Encode()` (skip2/go-qrcode) — we do not use the library's
  own image path. The authenticator app shows
  "Bolt Card Hub (hub.example.com)" using the `host_domain` setting as the
  account label.

New dependency footprint: `github.com/pquerna/otp` and its transitive
`github.com/boombuler/barcode` (small, pure Go). `go.mod` / `go.sum` must be
committed (CI fails otherwise).

## Login flow (stateless — no half-authenticated state)

Extend `adminApiLogin` (`web/admin_api.go`) to accept `{ password, code? }`:

1. Verify password via existing `verifyAdminPassword`. Wrong → `401
   { error: "invalid password" }`.
2. If `admin_totp_enabled != "Y"` → issue session (unchanged behavior).
3. If enabled and `code` is empty → `401 { error: "2fa required",
   totpRequired: true }`. **No session issued.**
4. If `code` is present, accept it if **either**:
   - it is a valid TOTP code for `admin_totp_secret` (±1 skew), **or**
   - it matches an unused entry in `admin_totp_recovery_hash`
     (bcrypt compare). On a recovery match, **remove that hash** from the JSON
     array and persist (single-use), then issue the session.
   - Otherwise → `401 { error: "invalid code", totpRequired: true }`.
5. On success, issue the session token exactly as today (24h cookie).

The password is re-sent together with the code, so there is **no intermediate
"pending 2FA" token** to store or expire. Brute force is bounded by Caddy's
existing 10 req/min/IP limit (a 6-digit code with ±1 skew = 3 valid values out
of 1,000,000 per window).

`adminApiAuthCheck` is unchanged — it only validates an already-issued session
cookie, which is only minted after the full factor(s) pass.

## New endpoints (all behind existing `adminApiAuth` session)

Registered in the `CreateHandler_AdminApi` switch in `web/admin_api.go`:

| Method + path | Body | Returns |
|---------------|------|---------|
| `GET  /admin/api/auth/2fa/status` | — | `{ enabled: bool, recoveryCodesRemaining: int }` |
| `POST /admin/api/auth/2fa/setup` | — | `{ secret, otpauthUri, qrPng }` — generates a **pending** secret (`enabled` stays `N`). Returns `400` if `admin_totp_enabled == "Y"` (must `disable` first) so an active secret is never clobbered |
| `POST /admin/api/auth/2fa/enable` | `{ code }` | Validates `code` against the pending secret; sets `enabled=Y`; generates and returns `{ recoveryCodes: [...] }` **once** (stores only their bcrypt hashes) |
| `POST /admin/api/auth/2fa/disable` | `{ password }` | Re-verifies password via `verifyAdminPassword`; clears all three keys |

Handler code lives in a new `web/admin_api_2fa.go` (mirrors the
`admin_api_*.go` per-domain convention). TOTP helpers (generate/validate/
recovery-code generate+hash) live in a new `web/totp.go`.

**Recovery codes:** generated as ~10 random codes (e.g. 10 hex/base32 chars,
formatted for readability), shown to the admin exactly once on enable, and
persisted only as bcrypt hashes. The UI must warn that they cannot be shown
again.

## CLI recovery

Add `case "DisableAdmin2FA"` to the switch in `cli.go` →
clears `admin_totp_enabled`, `admin_totp_secret`, and
`admin_totp_recovery_hash` via `Db_set_setting(... "")`. Documented usage:

```bash
docker exec -it card ./app DisableAdmin2FA
```

Also add it to the CLI Commands list in CLAUDE.md.

## Frontend (admin-ui React SPA)

- **`hooks/use-auth.tsx`:** `login()` gains an optional `code` argument and
  surfaces a `totpRequired` signal (e.g. a typed error or returned status) so
  the login page can reveal the code field without losing the entered password.
- **`pages/login.tsx`:** when `totpRequired` is signalled, reveal a 6-digit
  code input plus a "Use a recovery code instead" toggle, and resubmit
  password + code. Follows existing shadcn `Input` / `Alert` / `Button`
  patterns already in the file.
- **`pages/settings.tsx`:** new **Security** card — "Two-Factor
  Authentication":
  - Disabled state → "Enable" button → setup flow: show QR (`qrPng`) + the
    manual base32 secret → enter a code → confirm → show recovery codes once
    (with copy/download affordance + "save these now" warning).
  - Enabled state → "codes remaining: N" + "Disable" button → confirm with
    password.
  - Backed by the four `/admin/api/auth/2fa/*` endpoints.

## Testing

Web-package unit tests using existing helpers (`openTestApp`,
`setupAdminSession`), generating live codes with `totp.GenerateCode(secret,
time.Now())`:

- 2FA enabled + password only → `401 totpRequired`.
- 2FA enabled + valid TOTP code → session issued.
- 2FA enabled + invalid code → `401`.
- Recovery code logs in once, then is **consumed** (second use of the same
  code fails).
- `setup` → `enable` → `status` reflects `enabled: true`; `disable` clears all
  three keys and `status` returns `enabled: false`.
- Settings-list endpoint never exposes `admin_totp_secret`
  (redaction assertion).

## Versioning

Bump `Version` in `docker/card/build/build.go` `0.19.8 → 0.20.0` (SemVer
minor: new backward-compatible feature) and update the matching `Version:`
references in CLAUDE.md and the project memory file.

**Convention refinement** (to record in CLAUDE.md + memory): adopt semantic
versioning — feature PRs bump MINOR, bug-fix/maintenance PRs bump PATCH,
breaking changes bump MAJOR. This supersedes the prior "patch-bump every PR"
rule.

## File-change checklist

**Backend**
- `web/totp.go` (new) — generate/validate TOTP, generate + hash recovery codes.
- `web/admin_api_2fa.go` (new) — status / setup / enable / disable handlers.
- `web/admin_api.go` — route the four new endpoints; extend `adminApiLogin`
  with `code` + TOTP/recovery verification.
- `web/admin_api_settings.go` — add `_secret` to the redaction suffix check.
- `cli.go` — `DisableAdmin2FA` command.
- `go.mod` / `go.sum` — add `github.com/pquerna/otp`.
- `build/build.go` — version bump.

**Frontend**
- `admin-ui/src/hooks/use-auth.tsx` — `login(password, code?)` + `totpRequired`.
- `admin-ui/src/pages/login.tsx` — conditional code field + recovery toggle.
- `admin-ui/src/pages/settings.tsx` — Security / 2FA management card.

**Docs**
- `CLAUDE.md` — settings keys, CLI command, versioning convention.
- Project memory file — new facts + versioning convention.

**Tests**
- `web/admin_api_2fa_test.go` (new) — login + enrollment + recovery + redaction.
