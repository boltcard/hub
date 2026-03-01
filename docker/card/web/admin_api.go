package web

import (
	"card/db"
	"card/util"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// adminApiAuth is middleware that validates the admin session cookie.
// Returns 401 JSON on failure (not a redirect like the HTML admin handler).
func (app *App) adminApiAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		c, err := r.Cookie("admin_session_token")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, map[string]string{"error": "not authenticated"})
			return
		}

		adminSessionToken := db.Db_get_setting(app.db_conn, "admin_session_token")
		if subtle.ConstantTimeCompare([]byte(c.Value), []byte(adminSessionToken)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, map[string]string{"error": "invalid session"})
			return
		}

		sessionCreatedStr := db.Db_get_setting(app.db_conn, "admin_session_created")
		if sessionCreatedStr != "" {
			sessionCreated, err := strconv.ParseInt(sessionCreatedStr, 10, 64)
			if err != nil || time.Now().Unix()-sessionCreated > 24*60*60 {
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(w, map[string]string{"error": "session expired"})
				return
			}
		}

		next(w, r)
	}
}

// CreateHandler_AdminApi dispatches /admin/api/* requests.
func (app *App) CreateHandler_AdminApi() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store")

		path := r.URL.Path

		switch {
		// Auth endpoints (no session required)
		case path == "/admin/api/auth/check":
			app.adminApiAuthCheck(w, r)

		case path == "/admin/api/auth/login" && r.Method == "POST":
			app.adminApiLogin(w, r)

		case path == "/admin/api/auth/register" && r.Method == "POST":
			app.adminApiRegister(w, r)

		case path == "/admin/api/auth/logout" && r.Method == "POST":
			app.adminApiLogout(w, r)

		// Protected endpoints (session required)
		case path == "/admin/api/dashboard":
			app.adminApiAuth(app.adminApiDashboard)(w, r)

		case path == "/admin/api/phoenix":
			app.adminApiAuth(app.adminApiPhoenix)(w, r)

		case path == "/admin/api/phoenix/transactions":
			app.adminApiAuth(app.adminApiTransactions)(w, r)

		case path == "/admin/api/cards" && r.Method == "GET":
			app.adminApiAuth(app.adminApiListCards)(w, r)

		case strings.HasPrefix(path, "/admin/api/cards/"):
			app.adminApiAuth(app.adminApiCardRouter)(w, r)

		case path == "/admin/api/settings" && r.Method == "GET":
			app.adminApiAuth(app.adminApiGetSettings)(w, r)

		case path == "/admin/api/settings/log-level" && r.Method == "PUT":
			app.adminApiAuth(app.adminApiSetLogLevel)(w, r)

		case path == "/admin/api/about" && r.Method == "GET":
			app.adminApiAuth(app.adminApiAbout)(w, r)

		case path == "/admin/api/about/update" && r.Method == "POST":
			app.adminApiAuth(app.adminApiTriggerUpdate)(w, r)

		case path == "/admin/api/database/stats" && r.Method == "GET":
			app.adminApiAuth(app.adminApiDatabaseStats)(w, r)

		case path == "/admin/api/database/download" && r.Method == "GET":
			app.adminApiAuth(app.adminApiDatabaseDownload)(w, r)

		case path == "/admin/api/database/import" && r.Method == "POST":
			app.adminApiAuth(app.adminApiDatabaseImport)(w, r)

		case path == "/admin/api/batch/create" && r.Method == "POST":
			app.adminApiAuth(app.adminApiBatchCreate)(w, r)

		default:
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"error": "not found"})
		}
	}
}

func (app *App) adminApiAuthCheck(w http.ResponseWriter, r *http.Request) {
	registered := db.Db_get_setting(app.db_conn, "admin_password_hash") != ""

	authenticated := false
	c, err := r.Cookie("admin_session_token")
	if err == nil {
		adminSessionToken := db.Db_get_setting(app.db_conn, "admin_session_token")
		if subtle.ConstantTimeCompare([]byte(c.Value), []byte(adminSessionToken)) == 1 {
			sessionCreatedStr := db.Db_get_setting(app.db_conn, "admin_session_created")
			if sessionCreatedStr != "" {
				sessionCreated, err := strconv.ParseInt(sessionCreatedStr, 10, 64)
				if err == nil && time.Now().Unix()-sessionCreated <= 24*60*60 {
					authenticated = true
				}
			}
		}
	}

	writeJSON(w, map[string]interface{}{
		"authenticated": authenticated,
		"registered":    registered,
	})
}

func (app *App) adminApiLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	adminPasswordHash := db.Db_get_setting(app.db_conn, "admin_password_hash")
	if adminPasswordHash == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "admin not registered"})
		return
	}

	valid := false
	if isBcryptHash(adminPasswordHash) {
		valid = CheckPassword(req.Password, adminPasswordHash)
	} else {
		// Legacy SHA256 path — check then migrate to bcrypt
		legacyHash := GetPwHash(app.db_conn, req.Password)
		if subtle.ConstantTimeCompare([]byte(legacyHash), []byte(adminPasswordHash)) == 1 {
			valid = true
			// Migrate to bcrypt
			newHash, err := HashPassword(req.Password)
			if err == nil {
				db.Db_set_setting(app.db_conn, "admin_password_hash", newHash)
				db.Db_set_setting(app.db_conn, "admin_password_salt", "")
			}
		}
	}

	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]string{"error": "invalid password"})
		return
	}

	sessionToken := util.Random_hex()
	db.Db_set_setting(app.db_conn, "admin_session_token", sessionToken)
	db.Db_set_setting(app.db_conn, "admin_session_created",
		strconv.FormatInt(time.Now().Unix(), 10))

	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session_token",
		Value:    sessionToken,
		Path:     "/admin/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, map[string]bool{"ok": true})
}

func (app *App) adminApiRegister(w http.ResponseWriter, r *http.Request) {
	if db.Db_get_setting(app.db_conn, "admin_password_hash") != "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "admin already registered"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "password required"})
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		log.Error("hash password error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "internal error"})
		return
	}

	db.Db_set_setting(app.db_conn, "admin_password_hash", hash)

	writeJSON(w, map[string]bool{"ok": true})
}

func (app *App) adminApiLogout(w http.ResponseWriter, r *http.Request) {
	ClearAdminSessionToken(w)
	db.Db_set_setting(app.db_conn, "admin_session_token", "")
	db.Db_set_setting(app.db_conn, "admin_session_created", "")
	writeJSON(w, map[string]bool{"ok": true})
}
