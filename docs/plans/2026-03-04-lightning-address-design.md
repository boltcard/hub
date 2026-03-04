# Lightning Address Support — Design

## Summary

Add lightning address support so each card has a randomly generated address (e.g. `a3f7b2c1@domain.com`) that anyone can pay to load funds onto the card. Addresses are auto-generated on card creation, visible on the admin card detail page with QR code, and can be toggled on/off per card.

## Database Changes (Schema v7)

Add two columns to `cards` table:

- `ln_address CHAR(12) NOT NULL DEFAULT ''` — random hex username (8 chars)
- `ln_address_enabled CHAR(1) NOT NULL DEFAULT 'Y'` — toggle

Partial unique index: `CREATE UNIQUE INDEX idx_cards_ln_address ON cards(ln_address) WHERE ln_address != ''`

Migration backfills existing cards with random hex values. New cards get one generated at insert time.

## LNURL-pay Protocol

Lightning address spec: `username@domain` resolves to `GET https://domain/.well-known/lnurlp/username`

### Endpoint 1: `GET /.well-known/lnurlp/{username}`

Metadata response. Looks up card by `ln_address` where `ln_address_enabled = 'Y'` and `wiped = 'N'`.

Response:
```json
{
  "tag": "payRequest",
  "callback": "https://{host_domain}/.well-known/lnurlp/{username}/callback",
  "minSendable": 1000,
  "maxSendable": 100000000000,
  "metadata": "[[\"text/plain\",\"Payment to {username}@{host_domain}\"]]",
  "commentAllowed": 140
}
```

Min 1 sat, max 100M sats (in millisats).

### Endpoint 2: `GET /.well-known/lnurlp/{username}/callback?amount={msats}&comment={text}`

Creates invoice. Same card lookup. Validates amount is between min/max sendable.

- Calls `phoenix.CreateInvoice()` with amount and metadata description hash
- Inserts `card_receipt` via `Db_add_card_receipt()` (starts unpaid)
- Returns `{"pr": "<bolt11>", "routes": []}`

Settlement: existing Phoenix listener calls `Db_set_receipt_paid()` by payment hash — no new code needed.

### Metadata Hash

Per LUD-06, the invoice description hash must be SHA256 of the metadata JSON string. The metadata string is `[[\"text/plain\",\"Payment to {username}@{host_domain}\"]]`.

## Admin UI Changes

Card detail page — new "Lightning Address" card between Info and Balance:

- Lightning address in monospace: `a3f7b2c1@domain.com`
- QR code encoding `lightning:a3f7b2c1@domain.com` (compact, wallet-compatible)
- Enable/Disable toggle
- Copy button

API changes:
- `GET /admin/api/cards/{id}` — add `lnAddress`, `lnAddressEnabled` fields
- `PUT /admin/api/cards/{id}/limits` — add `lnAddressEnabled` to request body

## Route Registration

In `app.go`, two new public routes (no auth):

```go
router.Path("/.well-known/lnurlp/{username}").Methods("GET").HandlerFunc(app.CreateHandler_LnurlpRequest())
router.Path("/.well-known/lnurlp/{username}/callback").Methods("GET").HandlerFunc(app.CreateHandler_LnurlpCallback())
```

Handler code in new file `web/lnurlp.go`.

## Rate Limiting

Add `/.well-known/lnurlp/*` to the Caddyfile `@api_paths` matcher (30 req/min per IP), same tier as other LNURL/payment endpoints.

## Files Changed

**Go backend:**
- `db/db_create.go` — `update_schema_6()` migration
- `db/db_init.go` — bump schema version to 7, call migration
- `db/db_get.go` — `Db_get_card_by_ln_address()`, add fields to `Card` struct, update `Db_get_card` scan
- `db/db_insert.go` — generate `ln_address` in insert functions
- `db/db_update.go` — add `ln_address_enabled` to card update functions
- `web/app.go` — register two new routes
- `web/lnurlp.go` — new file, two handlers
- `web/admin_api_cards.go` — add fields to get/update endpoints
- `build/build.go` — bump version
- `Caddyfile` — add LNURL-pay paths to rate limit matcher

**Frontend:**
- `admin-ui/src/pages/card-detail.tsx` — lightning address card with QR, toggle, copy

**Tests:**
- `db/db_test.go` — schema migration, new DB functions
- `web/web_test.go` — LNURL-pay endpoint tests
