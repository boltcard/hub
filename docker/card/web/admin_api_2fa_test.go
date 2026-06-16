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
