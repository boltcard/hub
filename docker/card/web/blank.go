package web

import (
	"net/http"
)

func Blank(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(""))
}
