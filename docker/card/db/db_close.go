package db

import (
	"card/util"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Close(db *sql.DB) {

	sqlStatement := `PRAGMA optimize;`
	rows, err := db.Query(sqlStatement)
	util.CheckAndPanic(err)
	defer rows.Close()

	db.Close()
}
