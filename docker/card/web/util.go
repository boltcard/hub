package web

import (
	"card/db"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strings"

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

func renderContent(w http.ResponseWriter, request string) {

	// default to index.html
	if strings.HasSuffix(request, "/") {
		request = request + "index.html"
	}

	// only log page requests
	if strings.HasSuffix(request, ".html") {
		log.Info("page : ", request)
		template_path := strings.Replace(request, "/admin/", "/dist/pages/admin/", 1)
		w.Header().Add("Content-Type", "text/html")
		renderTemplate(w, template_path, nil)
		return
	}

	// everything except .html
	content, err := os.Open("/web-content" + request)

	if err != nil {
		log.Info(err.Error())
		Blank(w, nil)
		return
	}

	defer content.Close()

	// https://stackoverflow.com/questions/19911929/what-mime-type-should-i-use-for-javascript-source-map-files
	switch {
	case strings.HasSuffix(request, ".js"):
		w.Header().Add("Content-Type", "application/json")
	case strings.HasSuffix(request, ".css"):
		w.Header().Add("Content-Type", "text/css")
	case strings.HasSuffix(request, ".png"):
		w.Header().Add("Content-Type", "image/png")
	case strings.HasSuffix(request, ".jpg"):
		w.Header().Add("Content-Type", "image/jpeg")
	case strings.HasSuffix(request, ".map"):
		w.Header().Add("Content-Type", "application/json")
	default:
		log.Info("suffix not recognised : ", request)
		return
	}

	io.Copy(w, content)
}
