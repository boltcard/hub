package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"card/db"

	"github.com/pquerna/otp/totp"
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

func TestAdminApi2faStatus_CorruptRecoveryHash(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	// A corrupt/un-parseable value must not break the endpoint or report codes.
	db.Db_set_setting(app.db_write, "admin_totp_recovery_hash", "not-valid-json")

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/auth/2fa/status", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"recoveryCodesRemaining":0`) == false {
		t.Fatalf("expected recoveryCodesRemaining=0 on corrupt input, got %s", w.Body.String())
	}
}

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

func TestAdminApi2faEnable_WithoutSetup(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("POST", "/admin/api/auth/2fa/enable",
		strings.NewReader(`{"code":"123456"}`))
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without prior setup, got %d", w.Code)
	}
}

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

	w1 := postLogin(app, `{"password":"testpass","code":"`+recovery[0]+`"}`)
	if w1.Code != http.StatusOK {
		t.Fatalf("recovery first use: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	if len(app.loadRecoveryHashes()) != 9 {
		t.Fatalf("expected 9 codes remaining, got %d", len(app.loadRecoveryHashes()))
	}

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

func TestAdminLogin_2fa_WrongPasswordValidCode_Fails(t *testing.T) {
	app := openTestApp(t)
	secret, _ := enable2faForTest(t, app)
	code, _ := totp.GenerateCode(secret, time.Now())

	w := postLogin(app, `{"password":"wrongpass","code":"`+code+`"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password with valid code: expected 401, got %d", w.Code)
	}
	if db.Db_get_setting(app.db_read, "admin_session_token") != "" {
		t.Fatal("no session should be issued with wrong password")
	}
}
