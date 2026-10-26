package web

import (
	"bytes"
	"card/db"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// Regression tests for the empty-session auth bypass: after logout (or before
// the first login) the stored session token was "", and a request carrying an
// empty cookie "admin_session_token=" matched it, with the expiry check
// skipped because admin_session_created was also "".

// requestWithCookie builds a request with a raw Cookie header so an empty
// cookie value is sent exactly as an attacker would send it.
func requestWithCookie(method, path, cookie string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	if cookie != "" {
		r.Header.Set("Cookie", cookie)
	}
	return r
}

func serveAdminApi(app *App, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	app.CreateHandler_AdminApi().ServeHTTP(w, r)
	return w
}

func logoutAdmin(t *testing.T, app *App, token string) {
	t.Helper()
	w := serveAdminApi(app, requestWithCookie("POST", "/admin/api/auth/logout",
		"admin_session_token="+token))
	if w.Code != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d", w.Code)
	}
}

// protected endpoints an attacker would go for first
var protectedAdminRoutes = []struct{ method, path string }{
	{"GET", "/admin/api/settings"},
	{"GET", "/admin/api/cards"},
	{"POST", "/admin/api/database/download"},
	{"GET", "/admin/api/dashboard"},
}

func assertRejected(t *testing.T, app *App, cookie string) {
	t.Helper()
	for _, rt := range protectedAdminRoutes {
		w := serveAdminApi(app, requestWithCookie(rt.method, rt.path, cookie))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with cookie %q: expected 401, got %d", rt.method, rt.path, cookie, w.Code)
		}
	}
}

func TestAdminSession_EmptyCookieAfterLogout(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	logoutAdmin(t, app, token)

	assertRejected(t, app, "admin_session_token=")
	assertRejected(t, app, `admin_session_token=""`)
	assertRejected(t, app, "admin_session_token="+token) // old cookie is dead
}

func TestAdminSession_LogoutLeavesUnguessableToken(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	logoutAdmin(t, app, token)

	stored := db.Db_get_setting(app.db_read, "admin_session_token")
	if stored == "" || stored == token {
		t.Fatalf("expected a fresh non-empty token after logout, got %q", stored)
	}
	if got := db.Db_get_setting(app.db_read, "admin_session_created"); got != "" {
		t.Fatalf("expected admin_session_created cleared, got %q", got)
	}
}

func TestAdminSession_EmptyCookieRegisteredNeverLoggedIn(t *testing.T) {
	app := openTestApp(t)
	hash, _ := HashPassword("testpass")
	db.Db_set_setting(app.db_write, "admin_password_hash", hash)

	assertRejected(t, app, "admin_session_token=")
}

func TestAdminSession_EmptyCookieUnregistered(t *testing.T) {
	app := openTestApp(t)
	assertRejected(t, app, "admin_session_token=")
}

// A database that has both settings cleared (as an older hub's logout left
// them) must not grant access to an empty cookie.
func TestAdminSession_EmptyCookieLegacyClearedSettings(t *testing.T) {
	app := openTestApp(t)
	setupAdminSession(t, app)
	db.Db_set_setting(app.db_write, "admin_session_token", "")
	db.Db_set_setting(app.db_write, "admin_session_created", "")

	assertRejected(t, app, "admin_session_token=")
}

func TestAdminSession_CreatedTimeMustBeValid(t *testing.T) {
	now := time.Now().Unix()
	cases := map[string]string{
		"empty":   "",
		"garbage": "not-a-number",
		"expired": strconv.FormatInt(now-24*60*60-60, 10),
		"future":  strconv.FormatInt(now+3600, 10),
	}
	for name, created := range cases {
		t.Run(name, func(t *testing.T) {
			app := openTestApp(t)
			token := setupAdminSession(t, app)
			db.Db_set_setting(app.db_write, "admin_session_created", created)
			assertRejected(t, app, "admin_session_token="+token)
		})
	}
}

func TestAdminSession_ValidSessionAccepted(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	w := serveAdminApi(app, requestWithCookie("GET", "/admin/api/settings",
		"admin_session_token="+token))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with a valid session, got %d", w.Code)
	}
}

func TestAdminSession_AuthCheckEmptyCookieAfterLogout(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	logoutAdmin(t, app, token)

	w := serveAdminApi(app, requestWithCookie("GET", "/admin/api/auth/check",
		"admin_session_token="))
	var resp struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Authenticated {
		t.Fatal("expected authenticated=false for an empty cookie after logout")
	}
}

func TestAdminSession_WebsocketEmptyCookieAfterLogout(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	logoutAdmin(t, app, token)

	handler := app.CreateHandler_Websocket()
	for _, cookie := range []string{"", "admin_session_token=", "admin_session_token=" + token} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, requestWithCookie("GET", "/admin/api/websocket", cookie))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("websocket with cookie %q: expected 401, got %d", cookie, w.Code)
		}
	}
}

// importRequest builds a database import upload. The file is not a valid
// SQLite database, so a request that got past the password check would fail
// with 400 rather than overwrite /card_data/cards.db and exit the process.
func importRequest(t *testing.T, token, password string, withPassword bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if withPassword {
		mw.WriteField("password", password)
	}
	fw, err := mw.CreateFormFile("database_file", "cards.db")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("not a database"))
	mw.Close()

	r := httptest.NewRequest("POST", "/admin/api/database/import", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Cookie", "admin_session_token="+token)
	return r
}

func TestAdminDatabaseImport_RequiresPassword(t *testing.T) {
	app := openTestApp(t) // setupAdminSession sets the password to "testpass"
	token := setupAdminSession(t, app)

	if w := serveAdminApi(app, importRequest(t, token, "", false)); w.Code != http.StatusForbidden {
		t.Errorf("missing password: expected 403, got %d", w.Code)
	}
	if w := serveAdminApi(app, importRequest(t, token, "wrong", true)); w.Code != http.StatusForbidden {
		t.Errorf("wrong password: expected 403, got %d", w.Code)
	}
	// correct password reaches the import itself, which rejects the non-SQLite file
	if w := serveAdminApi(app, importRequest(t, token, "testpass", true)); w.Code != http.StatusBadRequest {
		t.Errorf("correct password: expected 400 (invalid SQLite file), got %d", w.Code)
	}
}

func TestAdminDatabaseImport_EmptyCookieAfterLogout(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	logoutAdmin(t, app, token)

	if w := serveAdminApi(app, importRequest(t, "", "testpass", true)); w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func jsonPost(path, token, body string) *http.Request {
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Cookie", "admin_session_token="+token)
	return r
}

// Allocating funds credits spendable balance, so it needs the password too.
func TestAdminAllocateFunds_RequiresPassword(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	cardId := insertFundedCard(t, app.db_write, 10000)
	path := "/admin/api/cards/" + strconv.Itoa(cardId) + "/allocate"

	for name, body := range map[string]string{
		"missing": `{"amountSats":5000}`,
		"wrong":   `{"amountSats":5000,"password":"wrong"}`,
	} {
		if w := serveAdminApi(app, jsonPost(path, token, body)); w.Code != http.StatusForbidden {
			t.Errorf("%s password: expected 403, got %d", name, w.Code)
		}
	}
	if got := db.Db_get_card_balance(app.db_read, cardId); got != 10000 {
		t.Fatalf("balance changed without a valid password: %d", got)
	}

	w := serveAdminApi(app, jsonPost(path, token, `{"amountSats":5000,"password":"testpass"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("correct password: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// A funded batch credits every card it dispenses, so it needs the password;
// an unfunded batch does not.
func TestAdminBatchCreate_FundedRequiresPassword(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	const path = "/admin/api/batch/create"

	for name, body := range map[string]string{
		"missing": `{"groupTag":"g","maxCards":5,"initialBalance":1000,"expiryHours":24}`,
		"wrong":   `{"groupTag":"g","maxCards":5,"initialBalance":1000,"expiryHours":24,"password":"wrong"}`,
	} {
		if w := serveAdminApi(app, jsonPost(path, token, body)); w.Code != http.StatusForbidden {
			t.Errorf("%s password: expected 403, got %d", name, w.Code)
		}
	}

	var batches int
	if err := app.db_read.QueryRow(`SELECT COUNT(*) FROM program_cards`).Scan(&batches); err != nil {
		t.Fatal(err)
	}
	if batches != 0 {
		t.Fatalf("expected no batch created without a valid password, got %d", batches)
	}

	if w := serveAdminApi(app, jsonPost(path, token,
		`{"groupTag":"g","maxCards":5,"initialBalance":1000,"expiryHours":24,"password":"testpass"}`)); w.Code != http.StatusOK {
		t.Errorf("funded with correct password: expected 200, got %d", w.Code)
	}
	if w := serveAdminApi(app, jsonPost(path, token,
		`{"groupTag":"g","maxCards":5,"initialBalance":0,"expiryHours":24}`)); w.Code != http.StatusOK {
		t.Errorf("unfunded without password: expected 200, got %d", w.Code)
	}
}

// A stored token of "" with a valid creation time (e.g. from an imported
// database) must still reject an empty cookie.
func TestAdminSession_EmptyStoredTokenValidCreated(t *testing.T) {
	app := openTestApp(t)
	setupAdminSession(t, app)
	db.Db_set_setting(app.db_write, "admin_session_token", "")

	assertRejected(t, app, "admin_session_token=")
}

// Logout without a valid session must not end the admin's live session.
func TestAdminLogout_WithoutSessionLeavesSessionAlone(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	for _, cookie := range []string{"", "admin_session_token=", "admin_session_token=wrong"} {
		w := serveAdminApi(app, requestWithCookie("POST", "/admin/api/auth/logout", cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("logout with cookie %q: expected 200, got %d", cookie, w.Code)
		}
	}
	if got := db.Db_get_setting(app.db_read, "admin_session_token"); got != token {
		t.Fatalf("unauthenticated logout changed the session token to %q", got)
	}
	w := serveAdminApi(app, requestWithCookie("GET", "/admin/api/settings", "admin_session_token="+token))
	if w.Code != http.StatusOK {
		t.Fatalf("admin session should still work, got %d", w.Code)
	}
}

func TestAdminDatabaseDownload_RequiresPassword(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)
	const path = "/admin/api/database/download"

	if w := serveAdminApi(app, jsonPost(path, token, `{}`)); w.Code != http.StatusForbidden {
		t.Errorf("missing password: expected 403, got %d", w.Code)
	}
	if w := serveAdminApi(app, jsonPost(path, token, `{"password":"wrong"}`)); w.Code != http.StatusForbidden {
		t.Errorf("wrong password: expected 403, got %d", w.Code)
	}
	if w := serveAdminApi(app, jsonPost(path, token, `not json`)); w.Code != http.StatusBadRequest {
		t.Errorf("bad body: expected 400, got %d", w.Code)
	}
	// correct password reaches the download itself (the test has no
	// /card_data/cards.db, so it fails there rather than with 401/403)
	w := serveAdminApi(app, jsonPost(path, token, `{"password":"testpass"}`))
	if w.Code == http.StatusForbidden || w.Code == http.StatusUnauthorized {
		t.Errorf("correct password: got %d", w.Code)
	}
	// the old GET route is gone
	if w := serveAdminApi(app, requestWithCookie("GET", path, "admin_session_token="+token)); w.Code != http.StatusNotFound {
		t.Errorf("GET download: expected 404, got %d", w.Code)
	}
}

func TestAdminDatabaseImport_NoMultipartBody(t *testing.T) {
	app := openTestApp(t)
	token := setupAdminSession(t, app)

	w := serveAdminApi(app, jsonPost("/admin/api/database/import", token, `{"password":"testpass"}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// With 2FA enabled but an empty secret (only possible via an imported or
// edited database), a code generated from the empty key must not log in.
func TestAdminLogin_2faEmptySecretRejected(t *testing.T) {
	app := openTestApp(t)
	setupAdminSession(t, app)
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "Y")
	db.Db_set_setting(app.db_write, "admin_totp_secret", "")

	code, err := totp.GenerateCode("", time.Now())
	if err != nil {
		t.Skip("otp library refuses an empty key: ", err)
	}
	w := serveAdminApi(app, jsonPost("/admin/api/auth/login", "",
		`{"password":"testpass","code":"`+code+`"}`))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if validateTotpCode("", code) {
		t.Fatal("validateTotpCode accepted a code for an empty secret")
	}
}
