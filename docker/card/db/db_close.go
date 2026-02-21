package db

import (
	"card/util"
	"database/sql"
)

func Close(db *sql.DB) {

	sqlStatement := `PRAGMA optimize;`
	rows, err := db.Query(sqlStatement)
	util.CheckAndPanic(err)
	defer rows.Close()

	db.Close()
}
