package web

import (
	"card/db"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db_conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db_conn.Close() })
	os.Setenv("HOST_DOMAIN", "test.example.com")
	db.Db_init(db_conn)
	return db_conn
}

func openTestApp(t *testing.T) *App {
	t.Helper()
	db_conn := openTestDB(t)
	return NewApp(db_conn)
}

// Test getBearerToken with valid Bearer token
func TestGetBearerToken_Valid(t *testing.T) {
	r := httptest.NewRequest("GET", "/balance", nil)
	r.Header.Set("Authorization", "Bearer mytoken123")
	w := httptest.NewRecorder()

	token, ok := getBearerToken(w, r)
	if !ok {
		t.Fatal("expected ok=true for valid bearer token")
	}
	if token != "mytoken123" {
		t.Fatalf("expected token 'mytoken123', got '%s'", token)
	}
}

// Test getBearerToken with missing Authorization header
func TestGetBearerToken_Missing(t *testing.T) {
	r := httptest.NewRequest("GET", "/balance", nil)
	w := httptest.NewRecorder()

	token, ok := getBearerToken(w, r)
	if ok {
		t.Fatal("expected ok=false for missing auth header")
	}
	if token != "" {
		t.Fatalf("expected empty token, got '%s'", token)
	}

	// Should have written an error response
	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	if err != nil {
		t.Fatal("expected JSON error response")
	}
	if errResp.Code != 1 {
		t.Fatalf("expected error code 1, got %d", errResp.Code)
	}
}

// Test getBearerToken with malformed (Basic) auth
func TestGetBearerToken_Malformed(t *testing.T) {
	r := httptest.NewRequest("GET", "/balance", nil)
	r.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()

	_, ok := getBearerToken(w, r)
	if ok {
		t.Fatal("expected ok=false for Basic auth header")
	}
}

// Test balance handler with missing auth header returns error (not panic)
func TestBalance_MissingAuth(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_Balance()

	r := httptest.NewRequest("GET", "/balance", nil)
	w := httptest.NewRecorder()

	// This should NOT panic
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	if err != nil {
		t.Fatal("expected JSON error response, got: ", w.Body.String())
	}
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth' error, got '%s'", errResp.Error)
	}
}

// Test balance handler with invalid token
func TestBalance_InvalidToken(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_Balance()

	r := httptest.NewRequest("GET", "/balance", nil)
	r.Header.Set("Authorization", "Bearer invalidtoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	if err != nil {
		t.Fatal("expected JSON error response")
	}
	if errResp.Code != 1 {
		t.Fatalf("expected error code 1, got %d", errResp.Code)
	}
}

// Test balance handler with valid token returns balance
func TestBalance_ValidToken(t *testing.T) {
	app := openTestApp(t)

	// The test data created by db_init includes a test card
	// We need to set an access token for it
	// Card 1 should exist from test data - set its access token
	db.Db_set_setting(app.db_conn, "bolt_card_hub_api", "enabled")

	// Insert a card and set its tokens
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "testlogin", "testpass")

	// Set tokens for the card
	err := db.Db_set_tokens(app.db_conn, "testlogin", "testpass", "testaccesstoken", "testrefreshtoken")
	if err != nil {
		t.Fatal("failed to set tokens: ", err)
	}

	handler := app.CreateHandler_Balance()

	r := httptest.NewRequest("GET", "/balance", nil)
	r.Header.Set("Authorization", "Bearer testaccesstoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var balResp BalanceResponse
	err = json.Unmarshal(w.Body.Bytes(), &balResp)
	if err != nil {
		t.Fatal("expected JSON balance response, got: ", w.Body.String())
	}
	// New card with no transactions should have 0 balance
	if balResp.BTC.AvailableBalance != 0 {
		t.Fatalf("expected balance 0, got %d", balResp.BTC.AvailableBalance)
	}
}

// Test auth handler with login and password
func TestAuth_LoginPassword(t *testing.T) {
	app := openTestApp(t)

	// Insert a card
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "authlogin", "authpass")

	handler := app.CreateHandler_Auth()

	body := `{"login":"authlogin","password":"authpass"}`
	r := httptest.NewRequest("POST", "/auth?type=auth", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var authResp AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &authResp)
	if err != nil {
		t.Fatal("expected JSON auth response, got: ", w.Body.String())
	}
	if authResp.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if authResp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
}

// Test auth handler with invalid credentials
func TestAuth_InvalidCredentials(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_Auth()

	body := `{"login":"badlogin","password":"badpass"}`
	r := httptest.NewRequest("POST", "/auth?type=auth", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	if err != nil {
		t.Fatal("expected JSON error response")
	}
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth' error, got '%s'", errResp.Error)
	}
}

// Test path traversal is rejected
func TestPathTraversal(t *testing.T) {
	// Note: RenderStaticContent tries to open files from /web-content/
	// which doesn't exist in test. The path validation should reject
	// traversal attempts before even trying to open the file.

	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../etc/passwd"},
		{"double traversal", "../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"embedded traversal", "public/../../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			RenderStaticContent(w, tt.path)
			// Should return empty/blank response, NOT file contents
			body := w.Body.String()
			if strings.Contains(body, "root:") {
				t.Fatalf("path traversal succeeded for %s", tt.path)
			}
		})
	}
}

// Test atomic balance query
func TestAtomicBalance(t *testing.T) {
	db_conn := openTestDB(t)

	// Insert a card
	db.Db_insert_card(db_conn, "k0", "k1", "k2", "k3", "k4", "ballogin", "balpass")

	// Get card_id - look it up via access token after setting tokens
	err := db.Db_set_tokens(db_conn, "ballogin", "balpass", "baltoken", "balrefresh")
	if err != nil {
		t.Fatal("failed to set tokens")
	}
	card_id := db.Db_get_card_id_from_access_token(db_conn, "baltoken")
	if card_id == 0 {
		t.Fatal("expected non-zero card_id")
	}

	// No transactions, balance should be 0
	balance := db.Db_get_card_balance(db_conn, card_id)
	if balance != 0 {
		t.Fatalf("expected balance 0, got %d", balance)
	}

	// Add a receipt
	db.Db_add_card_receipt(db_conn, card_id, "lnbc1...", "abc123", 1000)
	// Mark it paid
	db.Db_set_receipt_paid(db_conn, "abc123")

	balance = db.Db_get_card_balance(db_conn, card_id)
	if balance != 1000 {
		t.Fatalf("expected balance 1000, got %d", balance)
	}

	// Add a payment
	db.Db_add_card_payment(db_conn, card_id, 300, "lnbc2...")

	balance = db.Db_get_card_balance(db_conn, card_id)
	if balance != 700 {
		t.Fatalf("expected balance 700, got %d", balance)
	}
}
