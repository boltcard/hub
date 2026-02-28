package db

import (
	"database/sql"

	log "github.com/sirupsen/logrus"
)

func Close(db *sql.DB) {

	sqlStatement := `PRAGMA optimize;`
	rows, err := db.Query(sqlStatement)
	if err != nil {
		log.Error("db close optimize error: ", err)
	} else {
		rows.Close()
	}

	db.Close()
}
