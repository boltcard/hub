# Admin Login 2FA (TOTP) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional TOTP two-factor authentication to the single-admin login, with recovery codes and a CLI escape hatch.

**Architecture:** TOTP secret + bcrypt-hashed recovery codes live in the existing `settings` key-value table (no schema migration). `pquerna/otp` handles RFC 6238. Login becomes a stateless single request `{password, code?}`: password is verified first, then — only when `admin_totp_enabled == "Y"` — a TOTP or recovery code is required before a session is issued. Enrollment and disable run through new session-protected `/admin/api/auth/2fa/*` endpoints. A `DisableAdmin2FA` CLI command clears the keys for lost-authenticator recovery.

**Tech Stack:** Go 1.25 (CGo sqlite), `github.com/pquerna/otp`, `golang.org/x/crypto/bcrypt`, React 19 + Vite + shadcn/ui.

**Spec:** `docs/superpowers/specs/2026-06-15-admin-totp-2fa-design.md`

**Working branch:** `claude/admin-totp-2fa` (already checked out; carries the uncommitted CLAUDE.md SemVer note from brainstorming — it folds into Task 10's commit).

**Conventions reminder:**
- All Go commands run from `docker/card/`. Tests: `go test -race -count=1 ./web/` (sets `HOST_DOMAIN` via helpers).
- Frontend build (typecheck) from `docker/card/admin-ui/`:
  ```bash
  export NVM_DIR="/home/debian/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0 > /dev/null 2>&1
  npm run build
  ```
- Settings access: `db.Db_get_setting(conn, key)` / `db.Db_set_setting(conn, key, value)`. Reads use `app.db_read`, writes use `app.db_write`.

---

## File Structure

**Backend (create):**
- `docker/card/web/totp.go` — pure TOTP/recovery-code helpers (no DB).
- `docker/card/web/totp_test.go` — unit tests for the helpers.
- `docker/card/web/admin_api_2fa.go` — settings helpers (DB) + the four `/admin/api/auth/2fa/*` handlers.
- `docker/card/web/admin_api_2fa_test.go` — handler + login-flow tests.

**Backend (modify):**
- `docker/card/web/admin_api.go` — route the four new endpoints; add TOTP enforcement to `adminApiLogin`.
- `docker/card/web/admin_api_settings.go` — add `_secret` to the redaction suffix list.
- `docker/card/cli.go` — `DisableAdmin2FA` command.
- `docker/card/go.mod` / `go.sum` — add `github.com/pquerna/otp`.
- `docker/card/build/build.go` — version bump.

**Frontend (modify):**
- `docker/card/admin-ui/src/hooks/use-auth.tsx` — `login(password, code?)` + `TotpRequiredError`.
- `docker/card/admin-ui/src/pages/login.tsx` — conditional code field + recovery toggle.
- `docker/card/admin-ui/src/pages/settings.tsx` — render the new 2FA card.

**Frontend (create):**
- `docker/card/admin-ui/src/components/two-factor-card.tsx` — enable/disable/enrollment UI.

**Docs (modify):**
- `CLAUDE.md` — settings keys, CLI command (SemVer note already edited, uncommitted).

---

## Task 1: Add `pquerna/otp` + TOTP helper module

**Files:**
- Create: `docker/card/web/totp.go`
- Create: `docker/card/web/totp_test.go`
- Modify: `docker/card/go.mod`, `docker/card/go.sum`

- [ ] **Step 1: Add the dependency**

From `docker/card/`:
```bash
go get github.com/pquerna/otp@v1.4.0
go mod tidy
```
Expected: `go.mod` gains `github.com/pquerna/otp v1.4.0` and `go.sum` updates (also pulls indirect `github.com/boombuler/barcode`).

- [ ] **Step 2: Write the failing test**

Create `docker/card/web/totp_test.go`:
```go
package web

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestNewTotpKey_ProducesValidatableSecret(t *testing.T) {
	secret, url, err := newTotpKey("hub.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Fatalf("expected otpauth URI, got %q", url)
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !validateTotpCode(secret, code) {
		t.Fatal("freshly generated code should validate")
	}
	if validateTotpCode(secret, "000000") {
		t.Fatal("an arbitrary wrong code should not validate")
	}
}

func TestGenerateRecoveryCodes_HashesMatch(t *testing.T) {
	plain, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 10 || len(hashes) != 10 {
		t.Fatalf("expected 10 codes and 10 hashes, got %d and %d", len(plain), len(hashes))
	}
	for i := range plain {
		if !CheckPassword(plain[i], hashes[i]) {
			t.Fatalf("hash %d does not verify against its plaintext", i)
		}
	}
	// codes must be unique
	seen := map[string]bool{}
	for _, c := range plain {
		if seen[c] {
			t.Fatalf("duplicate recovery code %q", c)
		}
		seen[c] = true
	}
}

func TestMatchRecoveryCode(t *testing.T) {
	plain, hashes, err := generateRecoveryCodes(3)
	if err != nil {
		t.Fatal(err)
	}
	idx, ok := matchRecoveryCode(plain[1], hashes)
	if !ok || idx != 1 {
		t.Fatalf("expected match at index 1, got idx=%d ok=%v", idx, ok)
	}
	if _, ok := matchRecoveryCode("nope-not-a-code", hashes); ok {
		t.Fatal("a non-code should not match")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./web/ -run 'Totp|RecoveryCode' -v`
Expected: compile failure — `undefined: newTotpKey`, `validateTotpCode`, `generateRecoveryCodes`, `matchRecoveryCode`.

- [ ] **Step 4: Implement the helpers**

Create `docker/card/web/totp.go`:
```go
package web

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/pquerna/otp/totp"
)

// newTotpKey generates a fresh TOTP secret for the admin. accountName is the
// label shown in the authenticator app (we pass host_domain). Returns the
// base32 secret and the otpauth:// provisioning URI.
func newTotpKey(accountName string) (secret string, url string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Bolt Card Hub",
		AccountName: accountName,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// validateTotpCode checks a 6-digit code against the secret. totp.Validate
// already applies a ±1 time-step skew to tolerate clock drift.
func validateTotpCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// generateRecoveryCodes returns `count` single-use recovery codes (8 hex chars
// each, 32 bits of entropy) plus their bcrypt hashes. Only the hashes are
// persisted; the plaintext is shown to the admin exactly once.
func generateRecoveryCodes(count int) (plain []string, hashes []string, err error) {
	for i := 0; i < count; i++ {
		b := make([]byte, 4)
		if _, err = rand.Read(b); err != nil {
			return nil, nil, err
		}
		code := hex.EncodeToString(b)
		h, herr := HashPassword(code)
		if herr != nil {
			return nil, nil, herr
		}
		plain = append(plain, code)
		hashes = append(hashes, h)
	}
	return plain, hashes, nil
}

// matchRecoveryCode returns the index of the first hash that verifies against
// code, or ok=false if none match.
func matchRecoveryCode(code string, hashes []string) (int, bool) {
	for i, h := range hashes {
		if CheckPassword(code, h) {
			return i, true
		}
	}
	return -1, false
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./web/ -run 'Totp|RecoveryCode' -v`
Expected: PASS for all three tests.

- [ ] **Step 6: Commit**

```bash
git add docker/card/web/totp.go docker/card/web/totp_test.go docker/card/go.mod docker/card/go.sum
git commit -m "Add TOTP and recovery-code helpers"
```

---

## Task 2: Redact the TOTP secret in the settings list

**Files:**
- Modify: `docker/card/web/admin_api_settings.go:24-26`
- Test: `docker/card/web/admin_api_2fa_test.go` (new file; first test added here)

- [ ] **Step 1: Write the failing test**

Create `docker/card/web/admin_api_2fa_test.go`. Start with only the imports this first test needs; Tasks 4 and 6 add `"time"` and `"github.com/pquerna/otp/totp"` when their tests first use them (Go errors on unused imports, so don't add them early):
```go
package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"card/db"
)

func TestAdminApiSettings_RedactsTotpSecret(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	db.Db_set_setting(app.db_write, "admin_totp_secret", "SUPERSECRETBASE32")

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/settings", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "SUPERSECRETBASE32") {
		t.Fatal("settings response leaked the raw TOTP secret")
	}

	var resp struct {
		Settings []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range resp.Settings {
		if s.Name == "admin_totp_secret" {
			found = true
			if s.Value != "REDACTED" {
				t.Fatalf("expected REDACTED, got %q", s.Value)
			}
		}
	}
	if !found {
		t.Fatal("admin_totp_secret not present in settings list")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./web/ -run TestAdminApiSettings_RedactsTotpSecret -v`
Expected: FAIL — value is the raw secret, not `REDACTED`.

- [ ] **Step 3: Add the `_secret` suffix to the redaction guard**

In `docker/card/web/admin_api_settings.go`, change:
```go
		if strings.HasSuffix(s.Name, "_hash") ||
			strings.HasSuffix(s.Name, "_token") ||
			strings.HasSuffix(s.Name, "_code") {
			value = "REDACTED"
```
to:
```go
		if strings.HasSuffix(s.Name, "_hash") ||
			strings.HasSuffix(s.Name, "_token") ||
			strings.HasSuffix(s.Name, "_code") ||
			strings.HasSuffix(s.Name, "_secret") {
			value = "REDACTED"
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./web/ -run TestAdminApiSettings_RedactsTotpSecret -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add docker/card/web/admin_api_settings.go docker/card/web/admin_api_2fa_test.go
git commit -m "Redact admin_totp_secret in settings list"
```

---

## Task 3: 2FA settings helpers + status endpoint

**Files:**
- Create: `docker/card/web/admin_api_2fa.go`
- Modify: `docker/card/web/admin_api.go` (route)
- Test: `docker/card/web/admin_api_2fa_test.go`

- [ ] **Step 1: Write the failing test**

Append to `docker/card/web/admin_api_2fa_test.go`:
```go
func TestAdminApi2faStatus_DisabledByDefault(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/auth/2fa/status", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Enabled                bool `json:"enabled"`
		RecoveryCodesRemaining int  `json:"recoveryCodesRemaining"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Enabled {
		t.Fatal("expected enabled=false by default")
	}
	if resp.RecoveryCodesRemaining != 0 {
		t.Fatalf("expected 0 remaining, got %d", resp.RecoveryCodesRemaining)
	}
}

func TestAdminApi2faStatus_RequiresSession(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/auth/2fa/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./web/ -run TestAdminApi2faStatus -v`
Expected: FAIL — route returns 404 (`adminApi2faStatus` not wired / not defined).

- [ ] **Step 3: Create the handler file with helpers + status handler**

Create `docker/card/web/admin_api_2fa.go` (the `card/util` import is added in Task 4 when the setup handler first needs it — adding it now would be an unused-import error):
```go
package web

import (
	"card/db"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// totpEnabled reports whether admin TOTP 2FA is currently active.
func (app *App) totpEnabled() bool {
	return db.Db_get_setting(app.db_read, "admin_totp_enabled") == "Y"
}

// loadRecoveryHashes returns the stored bcrypt hashes of unused recovery codes.
func (app *App) loadRecoveryHashes() []string {
	raw := db.Db_get_setting(app.db_read, "admin_totp_recovery_hash")
	if raw == "" {
		return nil
	}
	var hashes []string
	if err := json.Unmarshal([]byte(raw), &hashes); err != nil {
		log.Error("recovery hash unmarshal error: ", err)
		return nil
	}
	return hashes
}

// saveRecoveryHashes persists the recovery-code hashes as a JSON array.
func (app *App) saveRecoveryHashes(hashes []string) {
	b, err := json.Marshal(hashes)
	if err != nil {
		log.Error("recovery hash marshal error: ", err)
		return
	}
	db.Db_set_setting(app.db_write, "admin_totp_recovery_hash", string(b))
}

// consumeRecoveryCode returns true if code matches an unused recovery code,
// removing it (single use) before returning.
func (app *App) consumeRecoveryCode(code string) bool {
	hashes := app.loadRecoveryHashes()
	idx, ok := matchRecoveryCode(code, hashes)
	if !ok {
		return false
	}
	hashes = append(hashes[:idx], hashes[idx+1:]...)
	app.saveRecoveryHashes(hashes)
	return true
}

func (app *App) adminApi2faStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"enabled":                app.totpEnabled(),
		"recoveryCodesRemaining": len(app.loadRecoveryHashes()),
	})
}
```

- [ ] **Step 4: Wire the route**

In `docker/card/web/admin_api.go`, inside the `CreateHandler_AdminApi` switch, add after the existing auth cases (e.g. just after the `auth/logout` case):
```go
		case path == "/admin/api/auth/2fa/status" && r.Method == "GET":
			app.adminApiAuth(app.adminApi2faStatus)(w, r)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./web/ -run TestAdminApi2faStatus -v`
Expected: PASS for both status tests.

- [ ] **Step 6: Commit**

```bash
git add docker/card/web/admin_api_2fa.go docker/card/web/admin_api.go docker/card/web/admin_api_2fa_test.go
git commit -m "Add 2FA status endpoint and settings helpers"
```

---

## Task 4: 2FA setup + enable endpoints

**Files:**
- Modify: `docker/card/web/admin_api_2fa.go`
- Modify: `docker/card/web/admin_api.go` (routes)
- Test: `docker/card/web/admin_api_2fa_test.go`

- [ ] **Step 1: Write the failing test**

First add the two imports this test introduces to the import block of `docker/card/web/admin_api_2fa_test.go`:
```go
	"time"

	"github.com/pquerna/otp/totp"
```
(`"time"` goes in the stdlib group; the `totp` import goes in its own group after `"card/db"`.)

Then append to `docker/card/web/admin_api_2fa_test.go`:
```go
func TestAdminApi2faSetupThenEnable(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	handler := app.CreateHandler_AdminApi()
	cookie := &http.Cookie{Name: "admin_session_token", Value: token}

	// setup → returns a secret + QR, leaves 2FA disabled
	rs := httptest.NewRequest("POST", "/admin/api/auth/2fa/setup", nil)
	rs.AddCookie(cookie)
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, rs)
	if ws.Code != http.StatusOK {
		t.Fatalf("setup: expected 200, got %d: %s", ws.Code, ws.Body.String())
	}
	var setup struct {
		Secret     string `json:"secret"`
		OtpauthUri string `json:"otpauthUri"`
		QrPng      string `json:"qrPng"`
	}
	if err := json.Unmarshal(ws.Body.Bytes(), &setup); err != nil {
		t.Fatal(err)
	}
	if setup.Secret == "" || setup.QrPng == "" {
		t.Fatal("expected non-empty secret and qrPng")
	}
	if app.totpEnabled() {
		t.Fatal("2FA must not be enabled until a code is confirmed")
	}

	// enable with a wrong code → 400 (session is valid; the code is not)
	rbad := httptest.NewRequest("POST", "/admin/api/auth/2fa/enable",
		strings.NewReader(`{"code":"000000"}`))
	rbad.AddCookie(cookie)
	wbad := httptest.NewRecorder()
	handler.ServeHTTP(wbad, rbad)
	if wbad.Code != http.StatusBadRequest {
		t.Fatalf("enable(bad code): expected 400, got %d", wbad.Code)
	}
	if app.totpEnabled() {
		t.Fatal("2FA must stay disabled after a wrong code")
	}

	// enable with a valid code → 200, returns recovery codes, 2FA active
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	rok := httptest.NewRequest("POST", "/admin/api/auth/2fa/enable",
		strings.NewReader(`{"code":"`+code+`"}`))
	rok.AddCookie(cookie)
	wok := httptest.NewRecorder()
	handler.ServeHTTP(wok, rok)
	if wok.Code != http.StatusOK {
		t.Fatalf("enable(valid): expected 200, got %d: %s", wok.Code, wok.Body.String())
	}
	var en struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}
	if err := json.Unmarshal(wok.Body.Bytes(), &en); err != nil {
		t.Fatal(err)
	}
	if len(en.RecoveryCodes) != 10 {
		t.Fatalf("expected 10 recovery codes, got %d", len(en.RecoveryCodes))
	}
	if !app.totpEnabled() {
		t.Fatal("2FA should be enabled after a valid code")
	}
}

func TestAdminApi2faSetup_RejectedWhenAlreadyEnabled(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "Y")

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("POST", "/admin/api/auth/2fa/setup", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when already enabled, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./web/ -run TestAdminApi2faSetup -v`
Expected: FAIL — routes 404 (`adminApi2faSetup`/`adminApi2faEnable` undefined).

- [ ] **Step 3: Implement setup + enable handlers**

First add `"card/util"` to the import block of `docker/card/web/admin_api_2fa.go` (the setup handler uses `util.QrPngBase64Encode`). Then append:
```go
func (app *App) adminApi2faSetup(w http.ResponseWriter, r *http.Request) {
	if app.totpEnabled() {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "2fa already enabled"})
		return
	}

	hostDomain := db.Db_get_setting(app.db_read, "host_domain")
	secret, url, err := newTotpKey(hostDomain)
	if err != nil {
		log.Error("totp generate error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "internal error"})
		return
	}

	// Store the pending secret; it is inert until enable sets enabled=Y.
	db.Db_set_setting(app.db_write, "admin_totp_secret", secret)
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "N")

	writeJSON(w, map[string]string{
		"secret":     secret,
		"otpauthUri": url,
		"qrPng":      util.QrPngBase64Encode(url),
	})
}

func (app *App) adminApi2faEnable(w http.ResponseWriter, r *http.Request) {
	if app.totpEnabled() {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "2fa already enabled"})
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	secret := db.Db_get_setting(app.db_read, "admin_totp_secret")
	if secret == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "no pending 2fa setup"})
		return
	}
	if !validateTotpCode(secret, req.Code) {
		// session is valid; only the submitted code is wrong → 400, not 401,
		// so the shared apiFetch does not mislabel it as "session expired".
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid code"})
		return
	}

	plain, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		log.Error("recovery code generation error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "internal error"})
		return
	}
	app.saveRecoveryHashes(hashes)
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "Y")

	writeJSON(w, map[string]interface{}{"recoveryCodes": plain})
}
```

- [ ] **Step 4: Wire the routes**

In `docker/card/web/admin_api.go`, after the `2fa/status` case add:
```go
		case path == "/admin/api/auth/2fa/setup" && r.Method == "POST":
			app.adminApiAuth(app.adminApi2faSetup)(w, r)

		case path == "/admin/api/auth/2fa/enable" && r.Method == "POST":
			app.adminApiAuth(app.adminApi2faEnable)(w, r)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./web/ -run TestAdminApi2faSetup -v`
Expected: PASS for both setup/enable tests.

- [ ] **Step 6: Commit**

```bash
git add docker/card/web/admin_api_2fa.go docker/card/web/admin_api.go docker/card/web/admin_api_2fa_test.go
git commit -m "Add 2FA setup and enable endpoints"
```

---

## Task 5: 2FA disable endpoint

**Files:**
- Modify: `docker/card/web/admin_api_2fa.go`
- Modify: `docker/card/web/admin_api.go` (route)
- Test: `docker/card/web/admin_api_2fa_test.go`

- [ ] **Step 1: Write the failing test**

Append to `docker/card/web/admin_api_2fa_test.go`:
```go
func TestAdminApi2faDisable(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app) // sets admin_password_hash for "testpass"
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "Y")
	db.Db_set_setting(app.db_write, "admin_totp_secret", "SOMESECRET")
	app.saveRecoveryHashes([]string{"h1", "h2"})

	handler := app.CreateHandler_AdminApi()
	cookie := &http.Cookie{Name: "admin_session_token", Value: token}

	// wrong password → 400, 2FA stays on
	rbad := httptest.NewRequest("POST", "/admin/api/auth/2fa/disable",
		strings.NewReader(`{"password":"wrong"}`))
	rbad.AddCookie(cookie)
	wbad := httptest.NewRecorder()
	handler.ServeHTTP(wbad, rbad)
	if wbad.Code != http.StatusBadRequest {
		t.Fatalf("disable(wrong pw): expected 400, got %d", wbad.Code)
	}
	if !app.totpEnabled() {
		t.Fatal("2FA should remain enabled after a wrong password")
	}

	// correct password → 200, all keys cleared
	rok := httptest.NewRequest("POST", "/admin/api/auth/2fa/disable",
		strings.NewReader(`{"password":"testpass"}`))
	rok.AddCookie(cookie)
	wok := httptest.NewRecorder()
	handler.ServeHTTP(wok, rok)
	if wok.Code != http.StatusOK {
		t.Fatalf("disable(correct pw): expected 200, got %d: %s", wok.Code, wok.Body.String())
	}
	if app.totpEnabled() {
		t.Fatal("2FA should be disabled")
	}
	if db.Db_get_setting(app.db_read, "admin_totp_secret") != "" {
		t.Fatal("secret should be cleared")
	}
	if len(app.loadRecoveryHashes()) != 0 {
		t.Fatal("recovery hashes should be cleared")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./web/ -run TestAdminApi2faDisable -v`
Expected: FAIL — route 404 (`adminApi2faDisable` undefined).

- [ ] **Step 3: Implement the disable handler**

Append to `docker/card/web/admin_api_2fa.go`:
```go
func (app *App) adminApi2faDisable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}
	if !app.verifyAdminPassword(req.Password) {
		// in-session re-auth failure → 400 (see enable handler rationale)
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid password"})
		return
	}

	db.Db_set_setting(app.db_write, "admin_totp_enabled", "N")
	db.Db_set_setting(app.db_write, "admin_totp_secret", "")
	db.Db_set_setting(app.db_write, "admin_totp_recovery_hash", "")

	writeJSON(w, map[string]bool{"ok": true})
}
```

- [ ] **Step 4: Wire the route**

In `docker/card/web/admin_api.go`, after the `2fa/enable` case add:
```go
		case path == "/admin/api/auth/2fa/disable" && r.Method == "POST":
			app.adminApiAuth(app.adminApi2faDisable)(w, r)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./web/ -run TestAdminApi2faDisable -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add docker/card/web/admin_api_2fa.go docker/card/web/admin_api.go docker/card/web/admin_api_2fa_test.go
git commit -m "Add 2FA disable endpoint"
```

---

## Task 6: Enforce TOTP at login

**Files:**
- Modify: `docker/card/web/admin_api.go` (`adminApiLogin`)
- Test: `docker/card/web/admin_api_2fa_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `docker/card/web/admin_api_2fa_test.go`. This helper enables 2FA for an app and returns the secret:
```go
func enable2faForTest(t *testing.T, app *App) (secret string, recovery []string) {
	t.Helper()
	hash, _ := HashPassword("testpass")
	db.Db_set_setting(app.db_write, "admin_password_hash", hash)

	s, _, err := newTotpKey("hub.example.com")
	if err != nil {
		t.Fatal(err)
	}
	db.Db_set_setting(app.db_write, "admin_totp_secret", s)
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "Y")

	plain, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		t.Fatal(err)
	}
	app.saveRecoveryHashes(hashes)
	return s, plain
}

func postLogin(app *App, body string) *httptest.ResponseRecorder {
	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("POST", "/admin/api/auth/login", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestAdminLogin_2faRequired_PasswordOnly(t *testing.T) {
	app := openTestApp(t)
	enable2faForTest(t, app)

	w := postLogin(app, `{"password":"testpass"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		TotpRequired bool `json:"totpRequired"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.TotpRequired {
		t.Fatal("expected totpRequired=true")
	}
	// no session token should be issued
	if db.Db_get_setting(app.db_read, "admin_session_token") != "" {
		t.Fatal("no session should be issued without a code")
	}
}

func TestAdminLogin_2fa_ValidCode(t *testing.T) {
	app := openTestApp(t)
	secret, _ := enable2faForTest(t, app)
	code, _ := totp.GenerateCode(secret, time.Now())

	w := postLogin(app, `{"password":"testpass","code":"`+code+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if db.Db_get_setting(app.db_read, "admin_session_token") == "" {
		t.Fatal("expected a session token to be issued")
	}
}

func TestAdminLogin_2fa_InvalidCode(t *testing.T) {
	app := openTestApp(t)
	enable2faForTest(t, app)

	w := postLogin(app, `{"password":"testpass","code":"000000"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminLogin_2fa_RecoveryCodeConsumedOnce(t *testing.T) {
	app := openTestApp(t)
	_, recovery := enable2faForTest(t, app)

	// first use → success
	w1 := postLogin(app, `{"password":"testpass","code":"`+recovery[0]+`"}`)
	if w1.Code != http.StatusOK {
		t.Fatalf("recovery first use: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	if len(app.loadRecoveryHashes()) != 9 {
		t.Fatalf("expected 9 codes remaining, got %d", len(app.loadRecoveryHashes()))
	}

	// reuse of the same code → 401
	w2 := postLogin(app, `{"password":"testpass","code":"`+recovery[0]+`"}`)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("recovery reuse: expected 401, got %d", w2.Code)
	}
}

func TestAdminLogin_NoTotp_PasswordOnlyStillWorks(t *testing.T) {
	app := openTestApp(t)
	hash, _ := HashPassword("testpass")
	db.Db_set_setting(app.db_write, "admin_password_hash", hash)

	w := postLogin(app, `{"password":"testpass"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with 2FA off, got %d: %s", w.Code, w.Body.String())
	}
}
```

(Wrong-password behavior with 2FA on is already covered by the existing `invalid password` login tests, so no separate test is added here.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./web/ -run 'TestAdminLogin_2fa|TestAdminLogin_NoTotp' -v`
Expected: FAIL — `TestAdminLogin_2faRequired_PasswordOnly` currently returns 200 and issues a session (no enforcement yet).

- [ ] **Step 3: Add TOTP enforcement to `adminApiLogin`**

In `docker/card/web/admin_api.go`, update the request struct and add the enforcement block. Change:
```go
	var req struct {
		Password string `json:"password"`
	}
```
to:
```go
	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
```
Then, immediately **after** the `verifyAdminPassword` failure check (after its closing `}`) and **before** `sessionToken := util.Random_hex()`, insert:
```go
	if app.totpEnabled() {
		if req.Code == "" {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, map[string]interface{}{
				"error":        "2fa code required",
				"totpRequired": true,
			})
			return
		}
		secret := db.Db_get_setting(app.db_read, "admin_totp_secret")
		if !validateTotpCode(secret, req.Code) && !app.consumeRecoveryCode(req.Code) {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, map[string]interface{}{
				"error":        "invalid code",
				"totpRequired": true,
			})
			return
		}
	}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./web/ -run 'TestAdminLogin_2fa|TestAdminLogin_NoTotp' -v`
Expected: PASS for all.

- [ ] **Step 5: Run the full web package to check for regressions**

Run: `go test -race -count=1 ./web/`
Expected: PASS (all existing + new tests).

- [ ] **Step 6: Commit**

```bash
git add docker/card/web/admin_api.go docker/card/web/admin_api_2fa_test.go
git commit -m "Require TOTP or recovery code at admin login when 2FA enabled"
```

---

## Task 7: `DisableAdmin2FA` CLI command

**Files:**
- Modify: `docker/card/cli.go`

- [ ] **Step 1: Add the command case**

In `docker/card/cli.go`, inside the `switch args[0]` in `processArgs`, add before `default:`:
```go
	case "DisableAdmin2FA":
		disableAdmin2FA(db_conn)
```

- [ ] **Step 2: Add the function**

Append to `docker/card/cli.go`:
```go
// DisableAdmin2FA clears admin TOTP 2FA. Recovery path for a lost
// authenticator: run via `docker exec -it card ./app DisableAdmin2FA`.
func disableAdmin2FA(db_conn *sql.DB) {
	db.Db_set_setting(db_conn, "admin_totp_enabled", "N")
	db.Db_set_setting(db_conn, "admin_totp_secret", "")
	db.Db_set_setting(db_conn, "admin_totp_recovery_hash", "")
	log.Info("admin 2FA disabled")
}
```

- [ ] **Step 3: Verify it builds and vets**

Run: `go build ./... && go vet ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add docker/card/cli.go
git commit -m "Add DisableAdmin2FA CLI command for lost-authenticator recovery"
```

---

## Task 8: Frontend — `login()` with TOTP support

**Files:**
- Modify: `docker/card/admin-ui/src/hooks/use-auth.tsx`

The shared `apiFetch` throws `AuthError` on any 401 and discards the body, so `login()` must do its own fetch to read `totpRequired` and the precise error message.

- [ ] **Step 1: Add `TotpRequiredError` and update `login`**

In `docker/card/admin-ui/src/hooks/use-auth.tsx`:

Add near the top (after imports):
```tsx
export class TotpRequiredError extends Error {
  constructor() {
    super("2fa required");
    this.name = "TotpRequiredError";
  }
}
```

Change the context type:
```tsx
  login: (password: string) => Promise<void>;
```
to:
```tsx
  login: (password: string, code?: string) => Promise<void>;
```

Replace the `login` implementation:
```tsx
  const login = async (password: string) => {
    await apiPost("/auth/login", { password });
    await refresh();
  };
```
with:
```tsx
  const login = async (password: string, code?: string) => {
    const res = await fetch("/admin/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password, code }),
    });
    if (res.ok) {
      await refresh();
      return;
    }
    const body = await res.json().catch(() => ({}) as { error?: string; totpRequired?: boolean });
    if (body.totpRequired) {
      throw new TotpRequiredError();
    }
    throw new Error(body.error || `HTTP ${res.status}`);
  };
```

(`code` undefined is dropped by `JSON.stringify`, so the backend sees no code — identical to the password-only case.)

- [ ] **Step 2: Verify it typechecks**

Run (from `docker/card/admin-ui/`):
```bash
export NVM_DIR="/home/debian/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0 > /dev/null 2>&1
npm run build
```
Expected: build succeeds (no TS errors).

- [ ] **Step 3: Commit**

```bash
git add docker/card/admin-ui/src/hooks/use-auth.tsx
git commit -m "Frontend: login() handles TOTP code and totpRequired signal"
```

---

## Task 9: Frontend — login page code field

**Files:**
- Modify: `docker/card/admin-ui/src/pages/login.tsx`

- [ ] **Step 1: Replace the login page**

Replace the body of `docker/card/admin-ui/src/pages/login.tsx` with:
```tsx
import { useState, type FormEvent } from "react";
import { useAuth, TotpRequiredError } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Zap } from "lucide-react";

export function LoginPage() {
  const { login } = useAuth();
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [totpRequired, setTotpRequired] = useState(false);
  const [useRecovery, setUseRecovery] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(password, totpRequired ? code : undefined);
    } catch (err) {
      if (err instanceof TotpRequiredError) {
        setTotpRequired(true);
        setError("");
      } else {
        setError(err instanceof Error ? err.message : "Login failed");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <a href="/" className="inline-block mx-auto">
            <Zap className="mx-auto h-8 w-8 text-primary" />
          </a>
          <CardTitle className="text-xl">
            <a href="/" className="no-underline text-foreground hover:text-primary transition-colors">
              Bolt Card Hub
            </a>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <Label htmlFor="password">Admin Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoFocus
                disabled={totpRequired}
              />
            </div>
            {totpRequired && (
              <div className="space-y-2">
                <Label htmlFor="code">
                  {useRecovery ? "Recovery Code" : "Authentication Code"}
                </Label>
                <Input
                  id="code"
                  type="text"
                  inputMode={useRecovery ? "text" : "numeric"}
                  autoComplete="one-time-code"
                  value={code}
                  onChange={(e) => setCode(e.target.value.trim())}
                  required
                  autoFocus
                  placeholder={useRecovery ? "8-character code" : "6-digit code"}
                />
                <button
                  type="button"
                  className="text-xs text-muted-foreground underline hover:text-foreground"
                  onClick={() => {
                    setUseRecovery((v) => !v);
                    setCode("");
                  }}
                >
                  {useRecovery
                    ? "Use authenticator code instead"
                    : "Use a recovery code instead"}
                </button>
              </div>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading
                ? "Logging in..."
                : totpRequired
                  ? "Verify"
                  : "Login"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
```

- [ ] **Step 2: Verify it typechecks**

Run (from `docker/card/admin-ui/`):
```bash
export NVM_DIR="/home/debian/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0 > /dev/null 2>&1
npm run build
```
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add docker/card/admin-ui/src/pages/login.tsx
git commit -m "Frontend: login page prompts for TOTP / recovery code"
```

---

## Task 10: Frontend — 2FA management card on Settings

**Files:**
- Create: `docker/card/admin-ui/src/components/two-factor-card.tsx`
- Modify: `docker/card/admin-ui/src/pages/settings.tsx`

- [ ] **Step 1: Create the 2FA card component**

Create `docker/card/admin-ui/src/components/two-factor-card.tsx`:
```tsx
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch, apiPost } from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

interface TwoFaStatus {
  enabled: boolean;
  recoveryCodesRemaining: number;
}

interface SetupData {
  secret: string;
  otpauthUri: string;
  qrPng: string;
}

export function TwoFactorCard() {
  const queryClient = useQueryClient();
  const { data } = useQuery({
    queryKey: ["2fa-status"],
    queryFn: () => apiFetch<TwoFaStatus>("/auth/2fa/status"),
  });

  const [setup, setSetup] = useState<SetupData | null>(null);
  const [code, setCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null);
  const [disablePassword, setDisablePassword] = useState("");
  const [showDisable, setShowDisable] = useState(false);

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ["2fa-status"] });

  const startSetup = useMutation({
    mutationFn: () => apiPost<SetupData>("/auth/2fa/setup"),
    onSuccess: (d) => {
      setSetup(d);
      setCode("");
    },
    onError: (err) => toast.error(err.message),
  });

  const enable = useMutation({
    mutationFn: () => apiPost<{ recoveryCodes: string[] }>("/auth/2fa/enable", { code }),
    onSuccess: (d) => {
      setSetup(null);
      setRecoveryCodes(d.recoveryCodes);
      invalidate();
      toast.success("Two-factor authentication enabled");
    },
    onError: (err) => toast.error(err.message),
  });

  const disable = useMutation({
    mutationFn: () => apiPost("/auth/2fa/disable", { password: disablePassword }),
    onSuccess: () => {
      setShowDisable(false);
      setDisablePassword("");
      invalidate();
      toast.success("Two-factor authentication disabled");
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Two-Factor Authentication
          {data?.enabled ? (
            <Badge>Enabled</Badge>
          ) : (
            <Badge variant="secondary">Disabled</Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {data?.enabled ? (
          <>
            <p className="text-sm text-muted-foreground">
              Login requires a code from your authenticator app.
              {" "}Recovery codes remaining: {data.recoveryCodesRemaining}.
            </p>
            <Button variant="destructive" onClick={() => setShowDisable(true)}>
              Disable 2FA
            </Button>
          </>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">
              Add a second factor (TOTP) to admin login using an authenticator app.
            </p>
            <Button
              onClick={() => startSetup.mutate()}
              disabled={startSetup.isPending}
            >
              {startSetup.isPending ? "Preparing..." : "Enable 2FA"}
            </Button>
          </>
        )}
      </CardContent>

      {/* Enrollment dialog: QR + confirm code */}
      <Dialog open={!!setup} onOpenChange={(o) => !o && setSetup(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Scan with your authenticator</DialogTitle>
          </DialogHeader>
          {setup && (
            <div className="space-y-4">
              <img
                src={`data:image/png;base64,${setup.qrPng}`}
                alt="TOTP QR code"
                className="mx-auto h-48 w-48"
              />
              <p className="text-xs text-muted-foreground break-all">
                Manual key: <span className="font-mono">{setup.secret}</span>
              </p>
              <div className="space-y-2">
                <Label htmlFor="enable-code">Enter the 6-digit code</Label>
                <Input
                  id="enable-code"
                  inputMode="numeric"
                  value={code}
                  onChange={(e) => setCode(e.target.value.trim())}
                  placeholder="123456"
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button
              onClick={() => enable.mutate()}
              disabled={enable.isPending || code.length === 0}
            >
              {enable.isPending ? "Verifying..." : "Confirm"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Recovery codes shown once */}
      <Dialog open={!!recoveryCodes} onOpenChange={(o) => !o && setRecoveryCodes(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Save your recovery codes</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Store these somewhere safe. Each can be used once if you lose your
            authenticator. They will not be shown again.
          </p>
          <div className="grid grid-cols-2 gap-2 font-mono text-sm">
            {recoveryCodes?.map((c) => (
              <span key={c} className="rounded bg-muted px-2 py-1">{c}</span>
            ))}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                navigator.clipboard?.writeText((recoveryCodes ?? []).join("\n"));
                toast.success("Copied");
              }}
            >
              Copy
            </Button>
            <Button onClick={() => setRecoveryCodes(null)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Disable confirmation */}
      <Dialog open={showDisable} onOpenChange={setShowDisable}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable two-factor authentication</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="disable-pw">Confirm admin password</Label>
            <Input
              id="disable-pw"
              type="password"
              value={disablePassword}
              onChange={(e) => setDisablePassword(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button
              variant="destructive"
              onClick={() => disable.mutate()}
              disabled={disable.isPending || disablePassword.length === 0}
            >
              {disable.isPending ? "Disabling..." : "Disable"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
```

- [ ] **Step 2: Render it on the Settings page**

In `docker/card/admin-ui/src/pages/settings.tsx`:

Add the import:
```tsx
import { TwoFactorCard } from "@/components/two-factor-card";
```

In the returned JSX, place the card just under the heading, before the settings table. Change:
```tsx
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      {data.settings.length === 0 ? (
```
to:
```tsx
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      <TwoFactorCard />

      {data.settings.length === 0 ? (
```

- [ ] **Step 3: Verify it typechecks/builds**

Run (from `docker/card/admin-ui/`):
```bash
export NVM_DIR="/home/debian/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0 > /dev/null 2>&1
npm run build
```
Expected: build succeeds. If `apiPost<SetupData>("/auth/2fa/setup")` with no body trips lint, it's fine — `apiPost` treats body as optional.

- [ ] **Step 4: Commit**

```bash
git add docker/card/admin-ui/src/components/two-factor-card.tsx docker/card/admin-ui/src/pages/settings.tsx
git commit -m "Frontend: 2FA management card on settings page"
```

---

## Task 11: Version bump + docs

**Files:**
- Modify: `docker/card/build/build.go`
- Modify: `CLAUDE.md` (CLI command list, settings keys; SemVer note already edited and currently uncommitted)

- [ ] **Step 1: Bump the version (SemVer minor — new feature)**

In `docker/card/build/build.go` change:
```go
var Version string = "0.19.8"
```
to:
```go
var Version string = "0.20.0"
```

- [ ] **Step 2: Update CLAUDE.md**

In `CLAUDE.md`:

(a) Sync the version reference in the `build/` architecture bullet:
```
- `build/` — Version string (currently "0.19.8"), date/time injected at build
```
to:
```
- `build/` — Version string (currently "0.20.0"), date/time injected at build
```

(b) Add `DisableAdmin2FA` to the CLI Commands code block:
```
./app WipeCard <card_id>
```
add a line after it:
```
./app DisableAdmin2FA
```

(c) In the **Settings** section's active-settings list, add:
```
- `admin_totp_enabled`, `admin_totp_secret`, `admin_totp_recovery_hash` — optional admin login 2FA (TOTP). `admin_totp_secret` (base32) and `admin_totp_recovery_hash` (JSON array of bcrypt-hashed single-use recovery codes) are redacted in the settings UI; `admin_totp_enabled` ("Y"/"N") gates enforcement at login. Cleared by the `DisableAdmin2FA` CLI command.
```

(d) In the **Authentication** section, under the Admin bullet, append:
```
Optional TOTP 2FA: when `admin_totp_enabled="Y"`, login also requires a 6-digit TOTP code or a single-use recovery code (see `web/admin_api_2fa.go`, `web/totp.go`).
```

(e) In the redaction note ("values with `_hash`, `_token`, `_code` suffixes shown as REDACTED"), add `_secret`:
```
sensitive values (`_hash`, `_token`, `_code`, `_secret` suffixes) redacted
```

- [ ] **Step 3: Final full-suite verification**

Run from `docker/card/`:
```bash
go vet ./... && go build ./... && go test -race -count=1 ./...
```
Expected: all PASS.

Run from `docker/card/admin-ui/`:
```bash
export NVM_DIR="/home/debian/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm use v22.22.0 > /dev/null 2>&1
npm run build
```
Expected: build succeeds.

- [ ] **Step 4: Commit (folds in the SemVer CLAUDE.md edit from brainstorming)**

```bash
git add docker/card/build/build.go CLAUDE.md
git commit -m "Bump version to 0.20.0 and document admin 2FA"
```

---

## Post-implementation

- [ ] Update the project memory file (`~/.claude/projects/-home-debian-hub/memory/MEMORY.md`) with: the 2FA settings keys, the `DisableAdmin2FA` recovery command, login-only enforcement, and the `pquerna/otp` dependency. (Required by CLAUDE.md "Memory File" convention.)
- [ ] Open the PR per the finishing-a-development-branch flow. Confirm CI is green (vet, build, test, govulncheck, frontend build, Docker builds).

---

## Manual verification checklist (after deploy to test hub)

These exercise paths unit tests can't (real authenticator app, browser):
1. Settings → Enable 2FA → scan QR with an authenticator → confirm code → recovery codes shown once.
2. Log out → log in: password accepted → prompted for code → correct code logs in.
3. Wrong code rejected; "Use a recovery code instead" accepts a recovery code once; reused recovery code rejected.
4. Settings shows decremented "recovery codes remaining" after a recovery login.
5. `docker exec -it card ./app DisableAdmin2FA` → next login is password-only.
6. Settings list shows `admin_totp_secret` as REDACTED.
