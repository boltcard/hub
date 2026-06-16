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
