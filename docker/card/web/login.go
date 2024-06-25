package web

import (
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	renderContent(w, r)
}
