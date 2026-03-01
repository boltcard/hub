package web

import (
	"card/db"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func setupAdminSession(t *testing.T, app *App) string {
	t.Helper()
	hash, _ := HashPassword("testpass")
	db.Db_set_setting(app.db_conn, "admin_password_hash", hash)
	token := "testtoken123"
	db.Db_set_setting(app.db_conn, "admin_session_token", token)
	db.Db_set_setting(app.db_conn, "admin_session_created",
		strconv.FormatInt(time.Now().Unix(), 10))
	return token
}

func TestAdminApiAuthCheck_NoPassword(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()

	r := httptest.NewRequest("GET", "/admin/api/auth/check", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Authenticated bool `json:"authenticated"`
		Registered    bool `json:"registered"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Registered {
		t.Fatal("expected registered=false when no password set")
	}
	if resp.Authenticated {
		t.Fatal("expected authenticated=false")
	}
}

func TestAdminApiAuthCheck_WithPassword(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()

	hash, _ := HashPassword("testpass")
	db.Db_set_setting(app.db_conn, "admin_password_hash", hash)

	r := httptest.NewRequest("GET", "/admin/api/auth/check", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp struct {
		Authenticated bool `json:"authenticated"`
		Registered    bool `json:"registered"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Registered {
		t.Fatal("expected registered=true when password set")
	}
	if resp.Authenticated {
		t.Fatal("expected authenticated=false without cookie")
	}
}

func TestAdminApiAuthMiddleware_NoCookie(t *testing.T) {
	app := openTestApp(t)
	handler := app.adminApiAuth(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]bool{"ok": true})
	})

	r := httptest.NewRequest("GET", "/admin/api/dashboard", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminApiAuthMiddleware_ValidCookie(t *testing.T) {
	app := openTestApp(t)

	token := "abc123def456"
	db.Db_set_setting(app.db_conn, "admin_session_token", token)
	db.Db_set_setting(app.db_conn, "admin_session_created",
		strconv.FormatInt(time.Now().Unix(), 10))

	handler := app.adminApiAuth(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]bool{"ok": true})
	})

	r := httptest.NewRequest("GET", "/admin/api/dashboard", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAdminApiAuthMiddleware_ExpiredSession(t *testing.T) {
	app := openTestApp(t)

	token := "abc123def456"
	db.Db_set_setting(app.db_conn, "admin_session_token", token)
	db.Db_set_setting(app.db_conn, "admin_session_created",
		strconv.FormatInt(time.Now().Unix()-25*60*60, 10))

	handler := app.adminApiAuth(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]bool{"ok": true})
	})

	r := httptest.NewRequest("GET", "/admin/api/dashboard", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminApiRegister(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()

	body := `{"password":"testpass123"}`
	r := httptest.NewRequest("POST", "/admin/api/auth/register",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	hash := db.Db_get_setting(app.db_conn, "admin_password_hash")
	if hash == "" {
		t.Fatal("expected password hash to be stored")
	}
	if !isBcryptHash(hash) {
		t.Fatal("expected bcrypt hash")
	}
}

func TestAdminApiRegister_AlreadyRegistered(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()

	hash, _ := HashPassword("existing")
	db.Db_set_setting(app.db_conn, "admin_password_hash", hash)

	body := `{"password":"newpass"}`
	r := httptest.NewRequest("POST", "/admin/api/auth/register",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminApiLogin_Success(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()

	hash, _ := HashPassword("correctpass")
	db.Db_set_setting(app.db_conn, "admin_password_hash", hash)

	body := `{"password":"correctpass"}`
	r := httptest.NewRequest("POST", "/admin/api/auth/login",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "admin_session_token" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected admin_session_token cookie")
	}

	token := db.Db_get_setting(app.db_conn, "admin_session_token")
	if token == "" {
		t.Fatal("expected session token in DB")
	}
}

func TestAdminApiLogin_WrongPassword(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()

	hash, _ := HashPassword("correctpass")
	db.Db_set_setting(app.db_conn, "admin_password_hash", hash)

	body := `{"password":"wrongpass"}`
	r := httptest.NewRequest("POST", "/admin/api/auth/login",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAdminApiLogout(t *testing.T) {
	app := openTestApp(t)
	handler := app.CreateHandler_AdminApi()

	db.Db_set_setting(app.db_conn, "admin_session_token", "sometoken")
	db.Db_set_setting(app.db_conn, "admin_session_created",
		strconv.FormatInt(time.Now().Unix(), 10))

	r := httptest.NewRequest("POST", "/admin/api/auth/logout", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: "sometoken"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	token := db.Db_get_setting(app.db_conn, "admin_session_token")
	if token != "" {
		t.Fatal("expected session token to be cleared")
	}
}

func TestAdminApiDashboard(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	insertFundedCard(t, app.db_conn, 50000)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/dashboard", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		CardCount int  `json:"cardCount"`
		HasCards  bool `json:"hasCards"`
		TopCards  []struct {
			CardId      int    `json:"cardId"`
			Note        string `json:"note"`
			BalanceSats int    `json:"balanceSats"`
		} `json:"topCards"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.CardCount != 1 {
		t.Fatalf("expected cardCount=1, got %d", resp.CardCount)
	}
	if !resp.HasCards {
		t.Fatal("expected hasCards=true")
	}
	if len(resp.TopCards) != 1 {
		t.Fatalf("expected 1 top card, got %d", len(resp.TopCards))
	}
	if resp.TopCards[0].BalanceSats != 50000 {
		t.Fatalf("expected balance 50000, got %d", resp.TopCards[0].BalanceSats)
	}
}

func TestAdminApiPhoenix(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/phoenix", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["channels"]; !ok {
		t.Fatal("expected channels field in response")
	}
}

func TestAdminApiSettings(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/settings", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Settings []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"settings"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	for _, s := range resp.Settings {
		if s.Name == "admin_password_hash" && s.Value != "REDACTED" {
			t.Fatal("expected admin_password_hash to be redacted")
		}
	}
}

func TestAdminApiSetLogLevel(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()

	body := `{"level":"debug"}`
	r := httptest.NewRequest("PUT", "/admin/api/settings/log-level",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	level := db.Db_get_setting(app.db_conn, "log_level")
	if level != "debug" {
		t.Fatalf("expected log_level=debug, got %s", level)
	}
}

func TestAdminApiListCards(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	insertFundedCard(t, app.db_conn, 50000)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/cards", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Cards []struct {
			CardId      int    `json:"cardId"`
			BalanceSats int    `json:"balanceSats"`
			Uid         string `json:"uid"`
		} `json:"cards"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(resp.Cards))
	}
	if resp.Cards[0].BalanceSats != 50000 {
		t.Fatalf("expected balance 50000, got %d", resp.Cards[0].BalanceSats)
	}
}

func TestAdminApiGetCard(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	cardId := insertFundedCard(t, app.db_conn, 75000)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/cards/"+strconv.Itoa(cardId), nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		CardId      int    `json:"cardId"`
		BalanceSats int    `json:"balanceSats"`
		Wiped       string `json:"wiped"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.CardId != cardId {
		t.Fatalf("expected cardId=%d, got %d", cardId, resp.CardId)
	}
	if resp.BalanceSats != 75000 {
		t.Fatalf("expected balance 75000, got %d", resp.BalanceSats)
	}
	if resp.Wiped != "N" {
		t.Fatalf("expected wiped=N, got %s", resp.Wiped)
	}
}

func TestAdminApiGetCard_NotFound(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/cards/99999", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAdminApiUpdateCardNote(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	cardId := insertFundedCard(t, app.db_conn, 10000)

	handler := app.CreateHandler_AdminApi()
	body := `{"note":"test card note"}`
	r := httptest.NewRequest("PUT", "/admin/api/cards/"+strconv.Itoa(cardId)+"/note",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify note was updated
	card, err := db.Db_get_card(app.db_conn, cardId)
	if err != nil {
		t.Fatal(err)
	}
	if card.Note != "test card note" {
		t.Fatalf("expected note='test card note', got '%s'", card.Note)
	}
}

func TestAdminApiUpdateCardLimits(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	cardId := insertFundedCard(t, app.db_conn, 10000)

	handler := app.CreateHandler_AdminApi()
	body := `{"txLimitSats":5000,"dayLimitSats":50000,"lnurlwEnable":"Y"}`
	r := httptest.NewRequest("PUT", "/admin/api/cards/"+strconv.Itoa(cardId)+"/limits",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify limits were updated
	card, err := db.Db_get_card(app.db_conn, cardId)
	if err != nil {
		t.Fatal(err)
	}
	if card.Tx_limit_sats != 5000 {
		t.Fatalf("expected tx_limit_sats=5000, got %d", card.Tx_limit_sats)
	}
	if card.Day_limit_sats != 50000 {
		t.Fatalf("expected day_limit_sats=50000, got %d", card.Day_limit_sats)
	}
	if card.Lnurlw_enable != "Y" {
		t.Fatalf("expected lnurlw_enable=Y, got %s", card.Lnurlw_enable)
	}
}

func TestAdminApiUpdateCardLimits_InvalidEnable(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	cardId := insertFundedCard(t, app.db_conn, 10000)

	handler := app.CreateHandler_AdminApi()
	body := `{"txLimitSats":5000,"dayLimitSats":50000,"lnurlwEnable":"X"}`
	r := httptest.NewRequest("PUT", "/admin/api/cards/"+strconv.Itoa(cardId)+"/limits",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminApiWipeCard(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	cardId := insertFundedCard(t, app.db_conn, 10000)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("POST", "/admin/api/cards/"+strconv.Itoa(cardId)+"/wipe", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Db_get_card filters wiped='N', so a wiped card returns error
	_, err := db.Db_get_card(app.db_conn, cardId)
	if err == nil {
		t.Fatal("expected error for wiped card (Db_get_card filters wiped='N')")
	}
}

func TestAdminApiCardTxs(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	cardId := insertFundedCard(t, app.db_conn, 50000)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/cards/"+strconv.Itoa(cardId)+"/txs", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Txs []struct {
			ReceiptId  int `json:"receiptId"`
			AmountSats int `json:"amountSats"`
		} `json:"txs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	// insertFundedCard creates one paid receipt
	if len(resp.Txs) != 1 {
		t.Fatalf("expected 1 tx, got %d", len(resp.Txs))
	}
	if resp.Txs[0].AmountSats != 50000 {
		t.Fatalf("expected amountSats=50000, got %d", resp.Txs[0].AmountSats)
	}
}

func TestAdminApiAbout(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/about", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Version         string `json:"version"`
		UpdateAvailable bool   `json:"updateAvailable"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Version == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestAdminApiBatchCreate(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	body := `{"groupTag":"test-batch","maxCards":5,"initialBalance":1000,"expiryHours":24}`
	r := httptest.NewRequest("POST", "/admin/api/batch/create",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Ok           bool   `json:"ok"`
		BoltcardLink string `json:"boltcardLink"`
		ProgramUrl   string `json:"programUrl"`
		Qr           string `json:"qr"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Ok {
		t.Fatal("expected ok=true")
	}
	if !strings.HasPrefix(resp.BoltcardLink, "boltcard://program?url=") {
		t.Fatalf("expected boltcard:// link, got %s", resp.BoltcardLink)
	}
	if !strings.Contains(resp.ProgramUrl, "/batch?s=") {
		t.Fatalf("expected program URL with /batch?s=, got %s", resp.ProgramUrl)
	}
	if resp.Qr == "" {
		t.Fatal("expected non-empty QR")
	}
}

func TestAdminApiBatchCreate_InvalidInput(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	body := `{"groupTag":"","maxCards":0,"initialBalance":0,"expiryHours":0}`
	r := httptest.NewRequest("POST", "/admin/api/batch/create",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminApiCardRouter_InvalidId(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	handler := app.CreateHandler_AdminApi()
	r := httptest.NewRequest("GET", "/admin/api/cards/abc", nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
