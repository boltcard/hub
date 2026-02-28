package web

import (
	"card/db"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
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
