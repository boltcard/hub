package web

import (
	"card/db"
	"card/phoenix"
	"card/util"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func getPwHash(passwordStr string) (passwordHashStr string) {
	passwordSalt := db.Db_get_setting("admin_password_salt")

	hasher := sha256.New()
	hasher.Write([]byte(passwordSalt))
	hasher.Write([]byte(passwordStr))
	passwordHash := hasher.Sum(nil)
	passwordHashStr = hex.EncodeToString(passwordHash)

	return passwordHashStr
}

func showDashboard(w http.ResponseWriter) {

	balance, err := phoenix.GetBalance()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	hostDomain := db.Db_get_setting("host_domain")
	gcUrl := db.Db_get_setting("gc_url")

	dashboardData := struct {
		QrValue      string
		FeeCreditSat int
		BalanceSat   int
		UpdateTime   string
	}{
		QrValue: `bluewallet:setlndhuburl?gc=https://` +
			gcUrl + `&url=https://` + hostDomain + `/`,
		FeeCreditSat: balance.FeeCreditSat,
		BalanceSat:   balance.BalanceSat,
		UpdateTime:   time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	renderTemplate(w, "dashboard", dashboardData)
}

func DashboardPage(w http.ResponseWriter, r *http.Request) {
	log.Info("dashboard Page request received")

	// detect if an admin password has been set
	if db.Db_get_setting("admin_password_hash") == "" {

		// handle admin_password_form postback
		if r.Method == "POST" {
			r.ParseForm()
			actionStr := r.Form.Get("action")
			if actionStr != "setup" {
				renderTemplate(w, "setup", nil)
				return
			}
			passwordStr := r.Form.Get("pw")
			passwordHashStr := getPwHash(passwordStr)
			db.Db_set_setting("admin_password_hash", passwordHashStr)
			renderTemplate(w, "login", nil)
			return
		}

		// return page for user to set an admin password
		renderTemplate(w, "setup", nil)
		return
	}

	// detect login postback
	if r.Method == "POST" {
		r.ParseForm()
		actionStr := r.Form.Get("action")

		if actionStr == "login" {
			pwStr := r.Form.Get("pw")
			pwHashStr := getPwHash(pwStr)
			adminPwHashStr := db.Db_get_setting("admin_password_hash")

			if pwHashStr == adminPwHashStr {

				//TODO: implement https://en.wikipedia.org/wiki/Post/Redirect/Get

				sessionToken := util.Random_hex()

				db.Db_set_setting("session_token", sessionToken)

				http.SetCookie(w, &http.Cookie{
					Name:    "session_token",
					Value:   sessionToken,
					Expires: time.Now().Add(24 * time.Hour),
				})

				showDashboard(w)
				return
			} else {
				// TODO: invalid login page
				w.Write([]byte("bad pw"))
				return
			}
		}

		if actionStr == "logout" {
			http.SetCookie(w, &http.Cookie{
				Name:    "session_token",
				Value:   "",
				Expires: time.Now(),
			})
			renderTemplate(w, "login", nil)
			return
		}

		renderTemplate(w, "login", nil)
		return
	}

	// detect if a session cookie exists
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// return page for user to log in as admin
			renderTemplate(w, "login", nil)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// validate the session cookie
	sessionToken := c.Value
	adminSessionToken := db.Db_get_setting("session_token")

	if sessionToken != adminSessionToken {
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   "",
			Expires: time.Now(),
		})
		renderTemplate(w, "login", nil)
		return
	}

	showDashboard(w)
}
