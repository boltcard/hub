package web

import (
	"card/db"
	"card/phoenix"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func StatusResponse(w http.ResponseWriter, r *http.Request) {
	// check that phoenix is running
	_, err := phoenix.GetBalance()
	if err != nil {
		log.Error(err)
		return
	}

	// check that the database is available
	_, err = db.Db_get_card_count()
	if err != nil {
		log.Error(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
