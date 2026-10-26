package web

import (
	"card/db"
	"net/http/httptest"
	"strings"
	"testing"
)

// Cards that never called /auth keep access_token = '' and refresh_token =
// ''. An empty bearer token or refresh token must never authenticate as them.

func TestWalletApi_EmptyBearerTokenRejected(t *testing.T) {
	app := setupEnabledApp(t)
	// funded card that has never authenticated
	db.Db_insert_card(app.db_write, "k0", "k1", "k2", "k3", "k4", "neverauthed", "pass")
	db.Db_add_card_receipt(app.db_write, 1, "inv", "h1", 50000)
	db.Db_set_receipt_paid(app.db_write, "h1", "test")

	for _, header := range []string{"Bearer ", "Bearer    ", "Bearer \t"} {
		r := httptest.NewRequest("GET", "/balance", nil)
		r.Header.Set("Authorization", header)
		w := httptest.NewRecorder()
		app.CreateHandler_Balance().ServeHTTP(w, r)

		if !strings.Contains(w.Body.String(), "Bad auth") {
			t.Errorf("header %q: expected Bad auth, got %d %s", header, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "50000") {
			t.Errorf("header %q: balance leaked: %s", header, w.Body.String())
		}
	}

	// keys must not be handed out (or rotated) either
	r := httptest.NewRequest("POST", "/getcardkeys", strings.NewReader(`{}`))
	r.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	app.CreateHandler_WalletApi_GetCardKeys().ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "Bad auth") {
		t.Errorf("getcardkeys: expected Bad auth, got %d %s", w.Code, w.Body.String())
	}
	card, err := db.Db_get_card(app.db_read, 1)
	if err != nil {
		t.Fatal(err)
	}
	if card.Key0_auth != "k0" {
		t.Errorf("card keys were rotated by an empty-token request")
	}
}

func TestWalletApi_EmptyRefreshTokenRejected(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_write, "k0", "k1", "k2", "k3", "k4", "neverauthed", "pass")

	r := httptest.NewRequest("POST", "/auth?type=refresh_token",
		strings.NewReader(`{"refresh_token":""}`))
	w := httptest.NewRecorder()
	app.CreateHandler_Auth().ServeHTTP(w, r)

	if !strings.Contains(w.Body.String(), "Bad auth") {
		t.Fatalf("expected Bad auth, got %d %s", w.Code, w.Body.String())
	}
}
