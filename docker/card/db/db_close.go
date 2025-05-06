package db

import (
	"card/util"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Close(db *sql.DB) {

	sqlStatement := `PRAGMA optimize;`
	_, err := db.Query(sqlStatement)
	util.CheckAndPanic(err)

	db.Close()
}
