package web

import (
	"card/db"
	"crypto/aes"
	"crypto/cipher"

	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aead/cmac"
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

// --- CompareVersions tests ---

func TestCompareVersions_Equal(t *testing.T) {
	if got := CompareVersions("1.2.3", "1.2.3"); got != 0 {
		t.Fatalf("expected 0 for equal versions, got %d", got)
	}
}

func TestCompareVersions_Newer(t *testing.T) {
	if got := CompareVersions("1.2.3", "1.2.4"); got != 1 {
		t.Fatalf("expected 1 for newer version, got %d", got)
	}
	if got := CompareVersions("1.2.3", "1.3.0"); got != 1 {
		t.Fatalf("expected 1 for newer minor, got %d", got)
	}
	if got := CompareVersions("1.2.3", "2.0.0"); got != 1 {
		t.Fatalf("expected 1 for newer major, got %d", got)
	}
}

func TestCompareVersions_Older(t *testing.T) {
	if got := CompareVersions("1.2.3", "1.2.2"); got != -1 {
		t.Fatalf("expected -1 for older version, got %d", got)
	}
	if got := CompareVersions("2.0.0", "1.9.9"); got != -1 {
		t.Fatalf("expected -1 for older major, got %d", got)
	}
}

func TestCompareVersions_MissingParts(t *testing.T) {
	// "1.2" treated as "1.2.0"
	if got := CompareVersions("1.2", "1.2.0"); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := CompareVersions("1.2", "1.2.1"); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestCompareVersions_SingleSegment(t *testing.T) {
	if got := CompareVersions("1", "2"); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	if got := CompareVersions("3", "1"); got != -1 {
		t.Fatalf("expected -1, got %d", got)
	}
}

// --- isBcryptHash tests ---

func TestIsBcryptHash_True(t *testing.T) {
	if !isBcryptHash("$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012") {
		t.Fatal("expected true for $2a$ prefix")
	}
	if !isBcryptHash("$2b$12$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012") {
		t.Fatal("expected true for $2b$ prefix")
	}
}

func TestIsBcryptHash_False(t *testing.T) {
	// SHA256 hex string
	if isBcryptHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855") {
		t.Fatal("expected false for SHA256 hex")
	}
	if isBcryptHash("plaintext") {
		t.Fatal("expected false for plaintext")
	}
}

// --- HashPassword + CheckPassword tests ---

func TestHashAndCheckPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("mysecretpassword")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !CheckPassword("mysecretpassword", hash) {
		t.Fatal("expected CheckPassword to return true for correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correctpassword")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if CheckPassword("wrongpassword", hash) {
		t.Fatal("expected CheckPassword to return false for wrong password")
	}
}

// --- Get_p_c tests ---

func TestGetPC_Valid(t *testing.T) {
	u, _ := url.Parse("https://example.com/ln?p=00112233445566778899aabbccddeeff&c=0011223344556677")
	p, c, err := Get_p_c(u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p) != 16 {
		t.Fatalf("expected p length 16, got %d", len(p))
	}
	if len(c) != 8 {
		t.Fatalf("expected c length 8, got %d", len(c))
	}
}

func TestGetPC_MissingP(t *testing.T) {
	u, _ := url.Parse("https://example.com/ln?c=0011223344556677")
	_, _, err := Get_p_c(u)
	if err == nil {
		t.Fatal("expected error for missing p")
	}
}

func TestGetPC_MissingC(t *testing.T) {
	u, _ := url.Parse("https://example.com/ln?p=00112233445566778899aabbccddeeff")
	_, _, err := Get_p_c(u)
	if err == nil {
		t.Fatal("expected error for missing c")
	}
}

func TestGetPC_NonHex(t *testing.T) {
	u, _ := url.Parse("https://example.com/ln?p=ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ&c=0011223344556677")
	_, _, err := Get_p_c(u)
	if err == nil {
		t.Fatal("expected error for non-hex p")
	}
}

func TestGetPC_WrongPLength(t *testing.T) {
	// p is only 8 bytes (16 hex chars) instead of 16 bytes (32 hex chars)
	u, _ := url.Parse("https://example.com/ln?p=0011223344556677&c=0011223344556677")
	_, _, err := Get_p_c(u)
	if err == nil {
		t.Fatal("expected error for wrong p length")
	}
}

func TestGetPC_WrongCLength(t *testing.T) {
	// c is 16 bytes instead of 8 bytes
	u, _ := url.Parse("https://example.com/ln?p=00112233445566778899aabbccddeeff&c=00112233445566778899aabbccddeeff")
	_, _, err := Get_p_c(u)
	if err == nil {
		t.Fatal("expected error for wrong c length")
	}
}

// --- getAuthenticatedCardID tests ---

func TestGetAuthenticatedCardID_Valid(t *testing.T) {
	app := openTestApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "validtoken", "refreshtoken")

	r := httptest.NewRequest("GET", "/balance", nil)
	r.Header.Set("Authorization", "Bearer validtoken")
	w := httptest.NewRecorder()

	cardId, ok := app.getAuthenticatedCardID(w, r)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if cardId == 0 {
		t.Fatal("expected non-zero card_id")
	}
}

func TestGetAuthenticatedCardID_InvalidToken(t *testing.T) {
	app := openTestApp(t)

	r := httptest.NewRequest("GET", "/balance", nil)
	r.Header.Set("Authorization", "Bearer badtoken")
	w := httptest.NewRecorder()

	cardId, ok := app.getAuthenticatedCardID(w, r)
	if ok {
		t.Fatal("expected ok=false for invalid token")
	}
	if cardId != 0 {
		t.Fatalf("expected card_id 0, got %d", cardId)
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Code != 1 {
		t.Fatalf("expected error code 1, got %d", errResp.Code)
	}
}

// --- /create handler tests ---

func setupEnabledApp(t *testing.T) *App {
	t.Helper()
	app := openTestApp(t)
	db.Db_set_setting(app.db_conn, "bolt_card_hub_api", "enabled")
	return app
}

func TestCreate_ValidInviteSecret(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_set_setting(app.db_conn, "invite_secret", "mysecret")

	handler := app.CreateHandler_Create()
	body := `{"invite_secret":"mysecret"}`
	r := httptest.NewRequest("POST", "/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp CreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Login == "" {
		t.Fatal("expected non-empty login")
	}
	if resp.Password == "" {
		t.Fatal("expected non-empty password")
	}
}

func TestCreate_WrongSecret(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_set_setting(app.db_conn, "invite_secret", "mysecret")

	handler := app.CreateHandler_Create()
	body := `{"invite_secret":"wrongsecret"}`
	r := httptest.NewRequest("POST", "/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth' error, got %q", errResp.Error)
	}
}

func TestCreate_EmptySecretMatchesNoSetting(t *testing.T) {
	app := setupEnabledApp(t)
	// No invite_secret set — defaults to empty string
	// Empty invite_secret in request should match

	handler := app.CreateHandler_Create()
	body := `{"invite_secret":""}`
	r := httptest.NewRequest("POST", "/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp CreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Login == "" {
		t.Fatal("expected non-empty login when no secret is configured")
	}
}

// --- /auth refresh flow tests ---

func TestAuthRefresh_ValidToken(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "access1", "refresh1")

	handler := app.CreateHandler_Auth()
	body := `{"refresh_token":"refresh1"}`
	r := httptest.NewRequest("POST", "/auth?type=refresh_token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.AccessToken == "" {
		t.Fatal("expected non-empty access token after refresh")
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token after refresh")
	}
	// Old tokens should be rotated
	if resp.RefreshToken == "refresh1" {
		t.Fatal("expected refresh token to be rotated")
	}
}

func TestAuthRefresh_InvalidToken(t *testing.T) {
	app := setupEnabledApp(t)

	handler := app.CreateHandler_Auth()
	body := `{"refresh_token":"nonexistent"}`
	r := httptest.NewRequest("POST", "/auth?type=refresh_token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth', got %q", errResp.Error)
	}
}

// --- /getcard handler tests ---

func TestGetCard_ValidToken(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "cardtoken", "cardrefresh")

	handler := app.CreateHandler_WalletApi_GetCard()
	r := httptest.NewRequest("POST", "/getcard", nil)
	r.Header.Set("Authorization", "Bearer cardtoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp CardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Status != "OK" {
		t.Fatalf("expected status OK, got %q", resp.Status)
	}
}

func TestGetCard_BadToken(t *testing.T) {
	app := setupEnabledApp(t)

	handler := app.CreateHandler_WalletApi_GetCard()
	r := httptest.NewRequest("POST", "/getcard", nil)
	r.Header.Set("Authorization", "Bearer invalidtoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth', got %q", errResp.Error)
	}
}

// --- /gettxs handler tests ---

func TestGetTxs_EmptyForNewCard(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "txtoken", "txrefresh")

	handler := app.CreateHandler_GetTxs()
	r := httptest.NewRequest("GET", "/gettxs", nil)
	r.Header.Set("Authorization", "Bearer txtoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var txs Transactions
	if err := json.Unmarshal(w.Body.Bytes(), &txs); err != nil {
		t.Fatalf("expected JSON array, got: %s", w.Body.String())
	}
	if len(txs) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txs))
	}
}

func TestGetTxs_ReturnsPayments(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "txtoken2", "txrefresh2")

	cardId := db.Db_get_card_id_from_access_token(app.db_conn, "txtoken2")
	db.Db_add_card_payment(app.db_conn, cardId, 100, "lnbc_pay1")
	db.Db_add_card_payment(app.db_conn, cardId, 200, "lnbc_pay2")

	handler := app.CreateHandler_GetTxs()
	r := httptest.NewRequest("GET", "/gettxs", nil)
	r.Header.Set("Authorization", "Bearer txtoken2")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var txs Transactions
	if err := json.Unmarshal(w.Body.Bytes(), &txs); err != nil {
		t.Fatalf("expected JSON array, got: %s", w.Body.String())
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}
}

// --- /updatecardwithpin handler tests ---

func TestUpdateCardWithPin_Valid(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "pintoken", "pinrefresh")

	handler := app.CreateHandler_WalletApi_UpdateCardWithPin()
	body := `{"enable":true,"card_name":"test","tx_max":"1000","day_max":"10000","enable_pin":true,"pin_limit_sats":"500","card_pin_number":"1234"}`
	r := httptest.NewRequest("POST", "/updatecardwithpin", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer pintoken")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp UpdateCardWithPinResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Status != "OK" {
		t.Fatalf("expected status OK, got %q", resp.Status)
	}

	// Verify the card was updated
	cardId := db.Db_get_card_id_from_access_token(app.db_conn, "pintoken")
	card, err := db.Db_get_card(app.db_conn, cardId)
	if err != nil {
		t.Fatal(err)
	}
	if card.Tx_limit_sats != 1000 {
		t.Fatalf("expected tx_limit_sats 1000, got %d", card.Tx_limit_sats)
	}
	if card.Day_limit_sats != 10000 {
		t.Fatalf("expected day_limit_sats 10000, got %d", card.Day_limit_sats)
	}
	if card.Pin_enable != "Y" {
		t.Fatalf("expected pin_enable Y, got %q", card.Pin_enable)
	}
	if card.Pin_number != "1234" {
		t.Fatalf("expected pin_number 1234, got %q", card.Pin_number)
	}
	if card.Lnurlw_enable != "Y" {
		t.Fatalf("expected lnurlw_enable Y, got %q", card.Lnurlw_enable)
	}
}

func TestUpdateCardWithPin_BadParams(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "pintoken2", "pinrefresh2")

	handler := app.CreateHandler_WalletApi_UpdateCardWithPin()
	// tx_max is not a number
	body := `{"enable":true,"card_name":"test","tx_max":"notanumber","day_max":"10000","enable_pin":false,"pin_limit_sats":"500"}`
	r := httptest.NewRequest("POST", "/updatecardwithpin", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer pintoken2")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "Bad param" {
		t.Fatalf("expected 'Bad param', got %q", errResp.Error)
	}
}

func TestUpdateCardWithPin_WithoutPin(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "pintoken3", "pinrefresh3")

	handler := app.CreateHandler_WalletApi_UpdateCardWithPin()
	// No card_pin_number — should use Db_update_card_without_pin
	body := `{"enable":false,"card_name":"test","tx_max":"500","day_max":"5000","enable_pin":false,"pin_limit_sats":"0"}`
	r := httptest.NewRequest("POST", "/updatecardwithpin", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer pintoken3")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp UpdateCardWithPinResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Status != "OK" {
		t.Fatalf("expected status OK, got %q", resp.Status)
	}
}

// --- /wipecard handler tests ---

func TestWipeCard_Success(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "aa00", "bb11", "cc22", "dd33", "ee44", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "wipetoken", "wiperefresh")

	handler := app.CreateHandler_WalletApi_WipeCard()
	r := httptest.NewRequest("POST", "/wipecard", nil)
	r.Header.Set("Authorization", "Bearer wipetoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp WipeCardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Status != "OK" {
		t.Fatalf("expected status OK, got %q", resp.Status)
	}
	if resp.Key0 != "aa00" {
		t.Fatalf("expected key0 aa00, got %q", resp.Key0)
	}
	if resp.Action != "wipe" {
		t.Fatalf("expected action wipe, got %q", resp.Action)
	}
}

func TestWipeCard_ThenAuthFails(t *testing.T) {
	app := setupEnabledApp(t)
	db.Db_insert_card(app.db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	db.Db_set_tokens(app.db_conn, "login1", "pass1", "wipetoken2", "wiperefresh2")

	// Wipe the card
	wipeHandler := app.CreateHandler_WalletApi_WipeCard()
	r := httptest.NewRequest("POST", "/wipecard", nil)
	r.Header.Set("Authorization", "Bearer wipetoken2")
	w := httptest.NewRecorder()
	wipeHandler.ServeHTTP(w, r)

	// Now try to use the same token — should fail
	balHandler := app.CreateHandler_Balance()
	r2 := httptest.NewRequest("GET", "/balance", nil)
	r2.Header.Set("Authorization", "Bearer wipetoken2")
	w2 := httptest.NewRecorder()
	balHandler.ServeHTTP(w2, r2)

	var errResp ErrorResponse
	json.Unmarshal(w2.Body.Bytes(), &errResp)
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth' after wipe, got %q", errResp.Error)
	}
}

// --- Feature flag gating tests ---

func TestFeatureFlag_HubApiDisabled(t *testing.T) {
	app := openTestApp(t)
	// bolt_card_hub_api is NOT set to "enabled"

	router := app.SetupRoutes()

	// Try to hit /create — should 405 (method not allowed) or 404
	r := httptest.NewRequest("POST", "/create", strings.NewReader(`{"invite_secret":""}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 when bolt_card_hub_api is disabled")
	}
}

func TestFeatureFlag_HubApiEnabled(t *testing.T) {
	app := openTestApp(t)
	db.Db_set_setting(app.db_conn, "bolt_card_hub_api", "enabled")

	router := app.SetupRoutes()

	// /create should be registered
	r := httptest.NewRequest("POST", "/create", strings.NewReader(`{"invite_secret":""}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Should get a 200 with a valid response (empty invite_secret matches no setting)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when hub API is enabled, got %d", w.Code)
	}

	var resp CreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Login == "" {
		t.Fatal("expected non-empty login")
	}
}

// === NFC Auth Chain Tests ===

// buildNfcTap constructs valid p (encrypted) and c (CMAC) values from raw keys.
func buildNfcTap(t *testing.T, key1, key2, uid []byte, counter uint32) (p, c []byte) {
	t.Helper()

	// Build plaintext: [0xC7, uid(7), counter(3 LE), 0x00(5)]
	plaintext := make([]byte, 16)
	plaintext[0] = 0xC7
	copy(plaintext[1:8], uid)
	plaintext[8] = byte(counter)
	plaintext[9] = byte(counter >> 8)
	plaintext[10] = byte(counter >> 16)

	// Encrypt with AES-CBC (zero IV) → p
	block, err := aes.NewCipher(key1)
	if err != nil {
		t.Fatal(err)
	}
	p = make([]byte, 16)
	cipher.NewCBCEncrypter(block, make([]byte, 16)).CryptBlocks(p, plaintext)

	// Build SV2 and compute truncated CMAC → c
	sv2 := make([]byte, 16)
	sv2[0] = 0x3c
	sv2[1] = 0xc3
	sv2[2] = 0x00
	sv2[3] = 0x01
	sv2[4] = 0x00
	sv2[5] = 0x80
	copy(sv2[6:13], uid)
	sv2[13] = byte(counter)
	sv2[14] = byte(counter >> 8)
	sv2[15] = byte(counter >> 16)

	c2, err := aes.NewCipher(key2)
	if err != nil {
		t.Fatal(err)
	}
	ks, err := cmac.Sum(sv2, c2, 16)
	if err != nil {
		t.Fatal(err)
	}
	c3, err := aes.NewCipher(ks)
	if err != nil {
		t.Fatal(err)
	}
	cm, err := cmac.Sum([]byte{}, c3, 16)
	if err != nil {
		t.Fatal(err)
	}
	c = []byte{cm[1], cm[3], cm[5], cm[7], cm[9], cm[11], cm[13], cm[15]}

	return p, c
}

var (
	nfcTestKey1 = []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	nfcTestKey2 = []byte{0xFF, 0xEE, 0xDD, 0xCC, 0xBB, 0xAA, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00}
	nfcTestUID  = []byte{0x04, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
)

func TestCheckCmac_Valid(t *testing.T) {
	_, validC := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 5)
	ctr := []byte{0x05, 0x00, 0x00}

	ok, err := check_cmac(nfcTestUID, ctr, nfcTestKey2, validC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected valid CMAC")
	}
}

func TestCheckCmac_Invalid(t *testing.T) {
	ctr := []byte{0x05, 0x00, 0x00}
	wrongC := make([]byte, 8) // all zeros

	ok, err := check_cmac(nfcTestUID, ctr, nfcTestKey2, wrongC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected invalid CMAC")
	}
}

func TestCheckCardTap_Valid(t *testing.T) {
	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 42)
	key1Hex := hex.EncodeToString(nfcTestKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)

	found, uidStr, counter := check_card_tap(p, c, key1Hex, key2Hex)
	if !found {
		t.Fatal("expected card to be found")
	}
	expectedUID := hex.EncodeToString(nfcTestUID)
	if uidStr != expectedUID {
		t.Fatalf("expected uid %s, got %s", expectedUID, uidStr)
	}
	if counter != 42 {
		t.Fatalf("expected counter 42, got %d", counter)
	}
}

func TestCheckCardTap_WrongKey(t *testing.T) {
	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 1)
	wrongKey1 := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00}
	wrongKey1Hex := hex.EncodeToString(wrongKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)

	found, _, _ := check_card_tap(p, c, wrongKey1Hex, key2Hex)
	if found {
		t.Fatal("expected card NOT to be found with wrong key1")
	}
}

func TestCheckCardTap_BadMagicByte(t *testing.T) {
	// Build plaintext with wrong magic byte, encrypt with correct key1
	plaintext := make([]byte, 16)
	plaintext[0] = 0xAA // not 0xC7
	copy(plaintext[1:8], nfcTestUID)
	plaintext[8] = 0x01

	block, _ := aes.NewCipher(nfcTestKey1)
	p := make([]byte, 16)
	cipher.NewCBCEncrypter(block, make([]byte, 16)).CryptBlocks(p, plaintext)

	// CMAC value doesn't matter since magic byte check fails first
	dummyC := make([]byte, 8)
	key1Hex := hex.EncodeToString(nfcTestKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)

	found, _, _ := check_card_tap(p, dummyC, key1Hex, key2Hex)
	if found {
		t.Fatal("expected card NOT to be found with bad magic byte")
	}
}

func TestCheckCardTap_WrongCmacKey(t *testing.T) {
	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 1)
	key1Hex := hex.EncodeToString(nfcTestKey1)
	// Pass wrong key2 — decryption succeeds but CMAC won't match
	wrongKey2 := []byte{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	wrongKey2Hex := hex.EncodeToString(wrongKey2)

	found, _, _ := check_card_tap(p, c, key1Hex, wrongKey2Hex)
	if found {
		t.Fatal("expected card NOT to be found with wrong key2")
	}
}

func TestFindCard_MatchesCorrectCard(t *testing.T) {
	db_conn := openTestDB(t)
	key1Hex := hex.EncodeToString(nfcTestKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)

	// Insert 3 cards; only the 2nd has matching keys
	db.Db_insert_card(db_conn, "k0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "k3", "k4", "login1", "pass1")
	db.Db_insert_card(db_conn, "k0", key1Hex, key2Hex, "k3", "k4", "login2", "pass2")
	db.Db_insert_card(db_conn, "k0", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "k3", "k4", "login3", "pass3")

	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 7)

	found, cardId, counter := Find_card(db_conn, p, c)
	if !found {
		t.Fatal("expected card to be found")
	}
	if cardId == 0 {
		t.Fatal("expected non-zero card_id")
	}
	if counter != 7 {
		t.Fatalf("expected counter 7, got %d", counter)
	}
}

func TestFindCard_NoMatch(t *testing.T) {
	db_conn := openTestDB(t)
	// Insert cards with keys that don't match the tap
	db.Db_insert_card(db_conn, "k0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "k3", "k4", "login1", "pass1")
	db.Db_insert_card(db_conn, "k0", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "k3", "k4", "login2", "pass2")

	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 1)

	found, _, _ := Find_card(db_conn, p, c)
	if found {
		t.Fatal("expected no card match")
	}
}

// === Admin Handler Tests ===

// The admin handler now serves the React SPA for all paths.
// Auth is handled by the SPA via /admin/api/auth/check.

func TestAdmin_ServesPage(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_Admin()

	// Without SPA files, handler returns 200 (blank page from Blank())
	r := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetPwHash_Deterministic(t *testing.T) {
	db_conn := openTestDB(t)
	db.Db_set_setting(db_conn, "admin_password_salt", "fixedsalt")

	hash1 := GetPwHash(db_conn, "password1")
	hash2 := GetPwHash(db_conn, "password1")
	if hash1 != hash2 {
		t.Fatal("expected same hash for same input")
	}

	hash3 := GetPwHash(db_conn, "password2")
	if hash1 == hash3 {
		t.Fatal("expected different hash for different password")
	}
}

// === LNURL Withdraw Flow Tests ===

// lnurlStatus is used to parse LNURL error/success JSON responses.
type lnurlStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// getPaidFlag queries the paid_flag for a given card_payment_id.
func getPaidFlag(db_conn *sql.DB, card_payment_id int) string {
	var flag string
	err := db_conn.QueryRow(
		`SELECT paid_flag FROM card_payments WHERE card_payment_id = $1`, card_payment_id,
	).Scan(&flag)
	if err != nil {
		return ""
	}
	return flag
}

// insertFundedCard inserts a card with the NFC test keys and funds it with the given balance.
func insertFundedCard(t *testing.T, db_conn *sql.DB, balanceSats int) int {
	t.Helper()
	key1Hex := hex.EncodeToString(nfcTestKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)
	db.Db_insert_card(db_conn, "k0", key1Hex, key2Hex, "k3", "k4", "lnlogin", "lnpass")
	err := db.Db_set_tokens(db_conn, "lnlogin", "lnpass", "lnaccess", "lnrefresh")
	if err != nil {
		t.Fatal("failed to set tokens:", err)
	}
	cardId := db.Db_get_card_id_from_access_token(db_conn, "lnaccess")
	if cardId == 0 {
		t.Fatal("expected non-zero card_id")
	}
	if balanceSats > 0 {
		db.Db_add_card_receipt(db_conn, cardId, "lnbc_fund", "fundhash", balanceSats)
		db.Db_set_receipt_paid(db_conn, "fundhash")
	}
	return cardId
}

// --- LnurlwRequest Handler Tests ---

func TestLnurlwRequest_ValidTap(t *testing.T) {
	app := openTestApp(t)
	key1Hex := hex.EncodeToString(nfcTestKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)
	db.Db_insert_card(app.db_conn, "k0", key1Hex, key2Hex, "k3", "k4", "lnlogin", "lnpass")
	db.Db_set_tokens(app.db_conn, "lnlogin", "lnpass", "lnaccess", "lnrefresh")
	cardId := db.Db_get_card_id_from_access_token(app.db_conn, "lnaccess")

	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 1)
	pHex := hex.EncodeToString(p)
	cHex := hex.EncodeToString(c)

	handler := app.CreateHandler_LnurlwRequest()
	r := httptest.NewRequest("GET", "/ln?p="+pHex+"&c="+cHex, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp LnurlwResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	if resp.Tag != "withdrawRequest" {
		t.Fatalf("expected tag 'withdrawRequest', got %q", resp.Tag)
	}
	if resp.Callback != "https://test.example.com/cb" {
		t.Fatalf("expected callback 'https://test.example.com/cb', got %q", resp.Callback)
	}
	if resp.Lnurlwk1 == "" {
		t.Fatal("expected non-empty k1")
	}
	if resp.MinWithdrawable != 1000 {
		t.Fatalf("expected minWithdrawable 1000, got %d", resp.MinWithdrawable)
	}
	if resp.MaxWithdrawable != 100_000_000_000 {
		t.Fatalf("expected maxWithdrawable 100000000000, got %d", resp.MaxWithdrawable)
	}

	// Verify counter was updated in DB
	newCounter := db.Db_get_card_counter(app.db_conn, cardId)
	if newCounter != 1 {
		t.Fatalf("expected counter 1, got %d", newCounter)
	}

	// Verify k1 was stored
	k1CardId, _ := db.Db_get_lnurlw_k1(app.db_conn, resp.Lnurlwk1)
	if k1CardId != cardId {
		t.Fatalf("expected k1 to map to card_id %d, got %d", cardId, k1CardId)
	}
}

func TestLnurlwRequest_BadParams(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_LnurlwRequest()

	tests := []struct {
		name string
		url  string
	}{
		{"missing both", "/ln"},
		{"missing c", "/ln?p=00112233445566778899aabbccddeeff"},
		{"invalid hex", "/ln?p=ZZZZ&c=0011223344556677"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			var resp lnurlStatus
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Status != "ERROR" || resp.Reason != "badly formatted request" {
				t.Fatalf("expected 'badly formatted request', got %q", resp.Reason)
			}
		})
	}
}

func TestLnurlwRequest_CardNotFound(t *testing.T) {
	app := openTestApp(t)
	// Insert a card with different keys so the tap won't match
	db.Db_insert_card(app.db_conn, "k0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "k3", "k4", "login1", "pass1")

	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 1)
	pHex := hex.EncodeToString(p)
	cHex := hex.EncodeToString(c)

	handler := app.CreateHandler_LnurlwRequest()
	r := httptest.NewRequest("GET", "/ln?p="+pHex+"&c="+cHex, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "card not found" {
		t.Fatalf("expected 'card not found', got %q", resp.Reason)
	}
}

func TestLnurlwRequest_CounterReplay(t *testing.T) {
	app := openTestApp(t)
	key1Hex := hex.EncodeToString(nfcTestKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)
	db.Db_insert_card(app.db_conn, "k0", key1Hex, key2Hex, "k3", "k4", "lnlogin", "lnpass")
	db.Db_set_tokens(app.db_conn, "lnlogin", "lnpass", "lnaccess", "lnrefresh")
	cardId := db.Db_get_card_id_from_access_token(app.db_conn, "lnaccess")

	// Set counter to 10 so a tap with counter=10 is a replay
	db.Db_set_card_counter(app.db_conn, cardId, 10)

	p, c := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 10)
	pHex := hex.EncodeToString(p)
	cHex := hex.EncodeToString(c)

	handler := app.CreateHandler_LnurlwRequest()
	r := httptest.NewRequest("GET", "/ln?p="+pHex+"&c="+cHex, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "card counter not incremented" {
		t.Fatalf("expected 'card counter not incremented', got %q", resp.Reason)
	}
}

func TestLnurlwRequest_CounterIncrement(t *testing.T) {
	app := openTestApp(t)
	key1Hex := hex.EncodeToString(nfcTestKey1)
	key2Hex := hex.EncodeToString(nfcTestKey2)
	db.Db_insert_card(app.db_conn, "k0", key1Hex, key2Hex, "k3", "k4", "lnlogin", "lnpass")
	db.Db_set_tokens(app.db_conn, "lnlogin", "lnpass", "lnaccess", "lnrefresh")
	cardId := db.Db_get_card_id_from_access_token(app.db_conn, "lnaccess")

	handler := app.CreateHandler_LnurlwRequest()

	// First tap with counter=5
	p1, c1 := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 5)
	r1 := httptest.NewRequest("GET", "/ln?p="+hex.EncodeToString(p1)+"&c="+hex.EncodeToString(c1), nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, r1)

	var resp1 LnurlwResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("first tap failed: %s", w1.Body.String())
	}
	if resp1.Tag != "withdrawRequest" {
		t.Fatalf("expected withdrawRequest, got %q", resp1.Tag)
	}

	ctr := db.Db_get_card_counter(app.db_conn, cardId)
	if ctr != 5 {
		t.Fatalf("expected counter 5 after first tap, got %d", ctr)
	}

	// Second tap with counter=6
	p2, c2 := buildNfcTap(t, nfcTestKey1, nfcTestKey2, nfcTestUID, 6)
	r2 := httptest.NewRequest("GET", "/ln?p="+hex.EncodeToString(p2)+"&c="+hex.EncodeToString(c2), nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)

	var resp2 LnurlwResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("second tap failed: %s", w2.Body.String())
	}
	if resp2.Tag != "withdrawRequest" {
		t.Fatalf("expected withdrawRequest on second tap, got %q", resp2.Tag)
	}

	ctr = db.Db_get_card_counter(app.db_conn, cardId)
	if ctr != 6 {
		t.Fatalf("expected counter 6 after second tap, got %d", ctr)
	}

	// k1 values should differ
	if resp1.Lnurlwk1 == resp2.Lnurlwk1 {
		t.Fatal("expected different k1 for each tap")
	}
}

// --- LnurlwCallback Pre-Validation Tests ---

// BOLT11 test invoice from ln-decodepay test suite (1,500 sats = 1,500,000 msats)
const testBolt11 = "lnbc15u1p3xnhl2pp5jptserfk3zk4qy42tlucycrfwxhydvlemu9pqr93tuzlv9cc7g3sdqsvfhkcap3xyhx7un8cqzpgxqzjcsp5f8c52y2stc300gl6s4xswtjpc37hrnnr3c9wvtgjfuvqmpm35evq9qyyssqy4lgd8tj637qcjp05rdpxxykjenthxftej7a2zzmwrmrl70fyj9hvj0rewhzj7jfyuwkwcg9g2jpwtk3wkjtwnkdks84hsnu8xps5vsq4gj5hs"

func setupK1(t *testing.T, db_conn *sql.DB, cardId int, k1 string, expiryOffset int64) {
	t.Helper()
	expiry := time.Now().Unix() + expiryOffset
	db.Db_set_lnurlw_k1(db_conn, cardId, k1, expiry)
}

func TestLnurlwCallback_MissingK1(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_LnurlwCallback()

	r := httptest.NewRequest("GET", "/cb", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "k1 not found" {
		t.Fatalf("expected 'k1 not found', got %q", resp.Reason)
	}
}

func TestLnurlwCallback_UnknownK1(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_LnurlwCallback()

	r := httptest.NewRequest("GET", "/cb?k1=deadbeef", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "card not found for k1 value" {
		t.Fatalf("expected 'card not found for k1 value', got %q", resp.Reason)
	}
}

func TestLnurlwCallback_ExpiredK1(t *testing.T) {
	app := openTestApp(t)
	cardId := insertFundedCard(t, app.db_conn, 5000)

	// Set k1 with past expiry
	setupK1(t, app.db_conn, cardId, "expiredk1", -60)

	handler := app.CreateHandler_LnurlwCallback()
	r := httptest.NewRequest("GET", "/cb?k1=expiredk1&pr="+testBolt11, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "k1 value expired" {
		t.Fatalf("expected 'k1 value expired', got %q", resp.Reason)
	}
}

func TestLnurlwCallback_InvalidInvoice(t *testing.T) {
	app := openTestApp(t)
	cardId := insertFundedCard(t, app.db_conn, 5000)
	setupK1(t, app.db_conn, cardId, "validk1", 300)

	handler := app.CreateHandler_LnurlwCallback()
	r := httptest.NewRequest("GET", "/cb?k1=validk1&pr=notaninvoice", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "invalid invoice" {
		t.Fatalf("expected 'invalid invoice', got %q", resp.Reason)
	}
}

func TestLnurlwCallback_InsufficientFunds(t *testing.T) {
	app := openTestApp(t)
	// Fund with 1000 sats, invoice is 1500 sats
	cardId := insertFundedCard(t, app.db_conn, 1000)
	setupK1(t, app.db_conn, cardId, "lowk1", 300)

	handler := app.CreateHandler_LnurlwCallback()
	r := httptest.NewRequest("GET", "/cb?k1=lowk1&pr="+testBolt11, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "Insufficient funds" {
		t.Fatalf("expected 'Insufficient funds', got %q", resp.Reason)
	}
}

func TestLnurlwCallback_InsufficientFundsWithFees(t *testing.T) {
	app := openTestApp(t)
	// Fund with 1505 sats; invoice=1500, fee headroom=4+1500*4/1000=10, total needed=1510
	cardId := insertFundedCard(t, app.db_conn, 1505)
	setupK1(t, app.db_conn, cardId, "feek1", 300)

	handler := app.CreateHandler_LnurlwCallback()
	r := httptest.NewRequest("GET", "/cb?k1=feek1&pr="+testBolt11, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Reason != "Insufficient funds with network fees" {
		t.Fatalf("expected 'Insufficient funds with network fees', got %q", resp.Reason)
	}
}

func TestLnurlwCallback_SufficientFundsReservesPayment(t *testing.T) {
	app := openTestApp(t)
	// Fund with 2000 sats; invoice=1500, fee headroom=10, total needed=1510 — passes
	cardId := insertFundedCard(t, app.db_conn, 2000)
	setupK1(t, app.db_conn, cardId, "goodk1", 300)

	handler := app.CreateHandler_LnurlwCallback()
	r := httptest.NewRequest("GET", "/cb?k1=goodk1&pr="+testBolt11, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// Phoenix is unavailable in tests → "no_config" → handlePaymentResult unlocks funds
	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ERROR" || resp.Reason != "phoenix config not set" {
		t.Fatalf("expected 'phoenix config not set' error, got status=%q reason=%q", resp.Status, resp.Reason)
	}

	// Verify payment record was created and then unlocked (paid_flag='N')
	var paidFlag string
	err := app.db_conn.QueryRow(
		`SELECT paid_flag FROM card_payments WHERE card_id = $1 ORDER BY card_payment_id DESC LIMIT 1`, cardId,
	).Scan(&paidFlag)
	if err != nil {
		t.Fatalf("failed to query payment: %v", err)
	}
	if paidFlag != "N" {
		t.Fatalf("expected paid_flag 'N' (unlocked), got %q", paidFlag)
	}
}

// --- handlePaymentResult Tests ---

func TestHandlePaymentResult_UnlocksFunds(t *testing.T) {
	cases := []struct {
		result string
	}{
		{"no_config"},
		{"failed_request_creation"},
		{"failed_read_response"},
	}
	for _, tc := range cases {
		t.Run(tc.result, func(t *testing.T) {
			db_conn := openTestDB(t)
			cardId := insertFundedCard(t, db_conn, 5000)
			paymentId := db.Db_add_card_payment(db_conn, cardId, 100, "lnbc_test")

			w := httptest.NewRecorder()
			returned := handlePaymentResult(w, db_conn, tc.result, paymentId)
			if !returned {
				t.Fatal("expected handlePaymentResult to return true")
			}

			// Should have written error JSON
			var resp lnurlStatus
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Status != "ERROR" {
				t.Fatalf("expected ERROR status, got %q", resp.Status)
			}

			// paid_flag should be 'N' (unlocked)
			if flag := getPaidFlag(db_conn, paymentId); flag != "N" {
				t.Fatalf("expected paid_flag 'N', got %q", flag)
			}
		})
	}
}

func TestHandlePaymentResult_KeepsFundsLocked(t *testing.T) {
	cases := []struct {
		result string
	}{
		{"phoenix_api_timeout"},
		{"fail_status_code"},
		{"failed_decode_response"},
		{"unknown_result"},
	}
	for _, tc := range cases {
		t.Run(tc.result, func(t *testing.T) {
			db_conn := openTestDB(t)
			cardId := insertFundedCard(t, db_conn, 5000)
			paymentId := db.Db_add_card_payment(db_conn, cardId, 100, "lnbc_test")

			w := httptest.NewRecorder()
			returned := handlePaymentResult(w, db_conn, tc.result, paymentId)
			if !returned {
				t.Fatal("expected handlePaymentResult to return true")
			}

			var resp lnurlStatus
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Status != "ERROR" {
				t.Fatalf("expected ERROR status, got %q", resp.Status)
			}

			// paid_flag should remain 'Y' (locked)
			if flag := getPaidFlag(db_conn, paymentId); flag != "Y" {
				t.Fatalf("expected paid_flag 'Y', got %q", flag)
			}
		})
	}
}

func TestHandlePaymentResult_NoError(t *testing.T) {
	db_conn := openTestDB(t)
	cardId := insertFundedCard(t, db_conn, 5000)
	paymentId := db.Db_add_card_payment(db_conn, cardId, 100, "lnbc_test")

	w := httptest.NewRecorder()
	returned := handlePaymentResult(w, db_conn, "no_error", paymentId)
	if returned {
		t.Fatal("expected handlePaymentResult to return false for 'no_error'")
	}

	// No JSON written
	if w.Body.Len() != 0 {
		t.Fatalf("expected no response body, got %q", w.Body.String())
	}

	// paid_flag should remain 'Y'
	if flag := getPaidFlag(db_conn, paymentId); flag != "Y" {
		t.Fatalf("expected paid_flag 'Y', got %q", flag)
	}
}

// --- handlePaymentReason Tests ---

func TestHandlePaymentReason_UnlocksFunds(t *testing.T) {
	cases := []struct {
		reason string
	}{
		{"this invoice has already been paid"},
		{"recipient node rejected the payment"},
		{"not enough funds in wallet to afford payment"},
		{"routing fees are insufficient"},
	}
	for _, tc := range cases {
		t.Run(tc.reason, func(t *testing.T) {
			db_conn := openTestDB(t)
			cardId := insertFundedCard(t, db_conn, 5000)
			paymentId := db.Db_add_card_payment(db_conn, cardId, 100, "lnbc_test")

			w := httptest.NewRecorder()
			returned := handlePaymentReason(w, db_conn, tc.reason, paymentId)
			if !returned {
				t.Fatal("expected handlePaymentReason to return true")
			}

			var resp lnurlStatus
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.Status != "ERROR" {
				t.Fatalf("expected ERROR status, got %q", resp.Status)
			}

			if flag := getPaidFlag(db_conn, paymentId); flag != "N" {
				t.Fatalf("expected paid_flag 'N', got %q", flag)
			}
		})
	}
}

func TestHandlePaymentReason_KeepsFundsLocked(t *testing.T) {
	db_conn := openTestDB(t)
	cardId := insertFundedCard(t, db_conn, 5000)
	paymentId := db.Db_add_card_payment(db_conn, cardId, 100, "lnbc_test")

	w := httptest.NewRecorder()
	returned := handlePaymentReason(w, db_conn, "some unknown reason", paymentId)
	if !returned {
		t.Fatal("expected handlePaymentReason to return true for unknown reason")
	}

	var resp lnurlStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ERROR" {
		t.Fatalf("expected ERROR status, got %q", resp.Status)
	}

	if flag := getPaidFlag(db_conn, paymentId); flag != "Y" {
		t.Fatalf("expected paid_flag 'Y', got %q", flag)
	}
}

func TestHandlePaymentReason_EmptyReason(t *testing.T) {
	db_conn := openTestDB(t)
	cardId := insertFundedCard(t, db_conn, 5000)
	paymentId := db.Db_add_card_payment(db_conn, cardId, 100, "lnbc_test")

	w := httptest.NewRecorder()
	returned := handlePaymentReason(w, db_conn, "", paymentId)
	if returned {
		t.Fatal("expected handlePaymentReason to return false for empty reason")
	}

	if w.Body.Len() != 0 {
		t.Fatalf("expected no response body, got %q", w.Body.String())
	}

	if flag := getPaidFlag(db_conn, paymentId); flag != "Y" {
		t.Fatalf("expected paid_flag 'Y', got %q", flag)
	}
}

// --- PayInvoice Handler Tests ---

func TestPayInvoice_MissingAuth(t *testing.T) {
	app := setupEnabledApp(t)
	handler := app.CreateHandler_WalletApi_PayInvoice()

	r := httptest.NewRequest("POST", "/payinvoice", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth', got %q", errResp.Error)
	}
}

func TestPayInvoice_InvalidJSON(t *testing.T) {
	app := setupEnabledApp(t)
	insertFundedCard(t, app.db_conn, 5000)
	handler := app.CreateHandler_WalletApi_PayInvoice()

	r := httptest.NewRequest("POST", "/payinvoice", strings.NewReader(`not json`))
	r.Header.Set("Authorization", "Bearer lnaccess")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Message != "request parameters invalid" {
		t.Fatalf("expected 'request parameters invalid', got %q", errResp.Message)
	}
}

func TestPayInvoice_InvalidInvoice(t *testing.T) {
	app := setupEnabledApp(t)
	insertFundedCard(t, app.db_conn, 5000)
	handler := app.CreateHandler_WalletApi_PayInvoice()

	body := `{"invoice":"notaninvoice","amount":100}`
	r := httptest.NewRequest("POST", "/payinvoice", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer lnaccess")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Message != "invalid invoice" {
		t.Fatalf("expected 'invalid invoice', got %q", errResp.Message)
	}
}

func TestPayInvoice_InsufficientBalance(t *testing.T) {
	app := setupEnabledApp(t)
	insertFundedCard(t, app.db_conn, 500) // only 500 sats
	handler := app.CreateHandler_WalletApi_PayInvoice()

	// testBolt11 is 1500 sats
	body := fmt.Sprintf(`{"invoice":"%s","amount":1500}`, testBolt11)
	r := httptest.NewRequest("POST", "/payinvoice", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer lnaccess")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Message != "invoice amount too large" {
		t.Fatalf("expected 'invoice amount too large', got %q", errResp.Message)
	}
}

func TestPayInvoice_DuplicateInvoice(t *testing.T) {
	app := setupEnabledApp(t)
	cardId := insertFundedCard(t, app.db_conn, 50000)
	handler := app.CreateHandler_WalletApi_PayInvoice()

	// Insert a paid payment with the same invoice
	db.Db_add_card_payment(app.db_conn, cardId, 1500, testBolt11)

	body := fmt.Sprintf(`{"invoice":"%s","amount":1500}`, testBolt11)
	r := httptest.NewRequest("POST", "/payinvoice", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer lnaccess")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Message != "invoice already paid" {
		t.Fatalf("expected 'invoice already paid', got %q", errResp.Message)
	}
}

// --- GetCardKeys Handler Tests ---

func TestGetCardKeys_Valid(t *testing.T) {
	app := setupEnabledApp(t)
	insertFundedCard(t, app.db_conn, 0)
	handler := app.CreateHandler_WalletApi_GetCardKeys()

	r := httptest.NewRequest("POST", "/getcardkeys", nil)
	r.Header.Set("Authorization", "Bearer lnaccess")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp CardKeysResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON, got: %s", w.Body.String())
	}
	if resp.ProtocolName != "create_bolt_card_response" {
		t.Fatalf("expected protocol_name 'create_bolt_card_response', got %q", resp.ProtocolName)
	}
	if resp.ProtocolVersion != 2 {
		t.Fatalf("expected protocol_version 2, got %d", resp.ProtocolVersion)
	}
	if resp.Key0 == "" || resp.Key1 == "" || resp.Key2 == "" || resp.Key3 == "" || resp.Key4 == "" {
		t.Fatal("expected non-empty keys")
	}
	if !strings.Contains(resp.LnurlwBase, "test.example.com") {
		t.Fatalf("expected lnurlw_base to contain host domain, got %q", resp.LnurlwBase)
	}
	// Verify keys changed from original
	card, err := db.Db_get_card(app.db_conn, 1)
	if err != nil {
		t.Fatal(err)
	}
	if card.Key0_auth == "k0" {
		t.Fatal("expected key0 to differ from original")
	}
}

func TestGetCardKeys_BadAuth(t *testing.T) {
	app := setupEnabledApp(t)
	handler := app.CreateHandler_WalletApi_GetCardKeys()

	r := httptest.NewRequest("POST", "/getcardkeys", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "Bad auth" {
		t.Fatalf("expected 'Bad auth', got %q", errResp.Error)
	}
}

// --- GetUserInvoices Handler Tests ---

func TestGetUserInvoices_Empty(t *testing.T) {
	app := setupEnabledApp(t)
	insertFundedCard(t, app.db_conn, 0)
	handler := app.CreateHandler_WalletApi_GetUserInvoices()

	r := httptest.NewRequest("GET", "/getuserinvoices", nil)
	r.Header.Set("Authorization", "Bearer lnaccess")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var invoices []UserInvoice
	if err := json.Unmarshal(w.Body.Bytes(), &invoices); err != nil {
		t.Fatalf("expected JSON array, got: %s", w.Body.String())
	}
	if len(invoices) != 0 {
		t.Fatalf("expected 0 invoices, got %d", len(invoices))
	}
}

func TestGetUserInvoices_WithReceipts(t *testing.T) {
	app := setupEnabledApp(t)
	cardId := insertFundedCard(t, app.db_conn, 0)
	handler := app.CreateHandler_WalletApi_GetUserInvoices()

	// Add 2 receipts: one paid, one unpaid
	db.Db_add_card_receipt(app.db_conn, cardId, "lnbc_paid", "paidhash", 500)
	db.Db_set_receipt_paid(app.db_conn, "paidhash")
	db.Db_add_card_receipt(app.db_conn, cardId, "lnbc_unpaid", "unpaidhash", 300)

	r := httptest.NewRequest("GET", "/getuserinvoices", nil)
	r.Header.Set("Authorization", "Bearer lnaccess")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var invoices []UserInvoice
	if err := json.Unmarshal(w.Body.Bytes(), &invoices); err != nil {
		t.Fatalf("expected JSON array, got: %s", w.Body.String())
	}
	if len(invoices) != 2 {
		t.Fatalf("expected 2 invoices, got %d", len(invoices))
	}

	foundPaid := false
	foundUnpaid := false
	for _, inv := range invoices {
		if inv.PaymentHash == "paidhash" {
			foundPaid = true
			if !inv.IsPaid {
				t.Fatal("expected paid receipt to have ispaid=true")
			}
			if inv.Amt != 500 {
				t.Fatalf("expected amt 500, got %d", inv.Amt)
			}
		}
		if inv.PaymentHash == "unpaidhash" {
			foundUnpaid = true
			if inv.IsPaid {
				t.Fatal("expected unpaid receipt to have ispaid=false")
			}
		}
	}
	if !foundPaid || !foundUnpaid {
		t.Fatal("expected both paid and unpaid receipts in response")
	}
}

// --- CreateCard / BCP Handler Tests ---

func TestCreateCard_Valid(t *testing.T) {
	app := openTestApp(t)
	db.Db_set_setting(app.db_conn, "new_card_code", "testsecret123")
	handler := app.CreateHandler_CreateCard()

	r := httptest.NewRequest("GET", "/new?a=testsecret123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp BcpResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON, got: %s", w.Body.String())
	}
	if resp.ProtocolName != "new_bolt_card_response" {
		t.Fatalf("expected protocol_name 'new_bolt_card_response', got %q", resp.ProtocolName)
	}
	if resp.K0 == "" || resp.K1 == "" || resp.K2 == "" || resp.K3 == "" || resp.K4 == "" {
		t.Fatal("expected non-empty keys")
	}
}

func TestCreateCard_MissingA(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_CreateCard()

	r := httptest.NewRequest("GET", "/new", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "a value not found") {
		t.Fatalf("expected 'a value not found', got %q", body)
	}
}

func TestCreateCard_WrongA(t *testing.T) {
	app := openTestApp(t)
	db.Db_set_setting(app.db_conn, "new_card_code", "correct")
	handler := app.CreateHandler_CreateCard()

	r := httptest.NewRequest("GET", "/new?a=wrong", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "a value not valid") {
		t.Fatalf("expected 'a value not valid', got %q", body)
	}
}

// --- BatchCreateCard Handler Tests ---

func TestBatchCreateCard_Valid(t *testing.T) {
	app := openTestApp(t)
	now := int(time.Now().Unix())
	db.Db_insert_program_cards(app.db_conn, "batchsecret", "group1", 10, 1000, now-60, now+3600)
	handler := app.CreateHandler_BatchCreateCard()

	body := `{"UID":"048B71B22D6B80"}`
	r := httptest.NewRequest("POST", "/batch?s=batchsecret", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	var resp BcpBatchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON, got: %s", w.Body.String())
	}
	if resp.K0 == "" || resp.K1 == "" || resp.K2 == "" || resp.K3 == "" || resp.K4 == "" {
		t.Fatal("expected non-empty keys")
	}
	if !strings.Contains(resp.Lnurlw, "test.example.com") {
		t.Fatalf("expected LNURLW to contain host domain, got %q", resp.Lnurlw)
	}
}

func TestBatchCreateCard_InvalidJSON(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_BatchCreateCard()

	r := httptest.NewRequest("POST", "/batch?s=whatever", strings.NewReader(`not json`))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBatchCreateCard_ExpiredProgram(t *testing.T) {
	app := openTestApp(t)
	now := int(time.Now().Unix())
	db.Db_insert_program_cards(app.db_conn, "expiredsecret", "group1", 10, 1000, now-7200, now-3600)
	handler := app.CreateHandler_BatchCreateCard()

	body := `{"UID":"048B71B22D6B80"}`
	r := httptest.NewRequest("POST", "/batch?s=expiredsecret", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "program card expired or not found") {
		t.Fatalf("expected 'program card expired or not found', got %q", w.Body.String())
	}
}

// --- PoS API Tests ---

func TestPosGetInfo_ReturnsEmpty(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_PosApi_GetInfo()

	r := httptest.NewRequest("GET", "/pos/getinfo", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "" {
		t.Fatalf("expected empty body, got %q", w.Body.String())
	}
}

func TestPosAddInvoice_InvalidJSON(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_PosApi_AddInvoice()

	r := httptest.NewRequest("POST", "/pos/addinvoice", strings.NewReader(`not json`))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPosAddInvoice_InvalidAmount(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_PosApi_AddInvoice()

	body := `{"Amt":"notanumber","Memo":"test"}`
	r := httptest.NewRequest("POST", "/pos/addinvoice", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid amount") {
		t.Fatalf("expected 'invalid amount', got %q", w.Body.String())
	}
}

func TestPosAddInvoice_NegativeAmount(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_PosApi_AddInvoice()

	body := `{"Amt":"-5","Memo":"test"}`
	r := httptest.NewRequest("POST", "/pos/addinvoice", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "amount must be positive") {
		t.Fatalf("expected 'amount must be positive', got %q", w.Body.String())
	}
}
