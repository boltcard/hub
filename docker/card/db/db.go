package db

import (
	"card/util"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Open() (*sql.DB, error) {

	// https://github.com/mattn/go-sqlite3
	// https://phiresky.github.io/blog/2020/sqlite-performance-tuning/

	// https://www.sqlite.org/pragma.html#pragma_journal_mode
	// https://www.sqlite.org/pragma.html#pragma_busy_timeout

	// WAL setting is not strictly needed as it is already set in Db_init()
	// busy_timeout is in ms

	db, err := sql.Open("sqlite3", "/card_data/cards.db?_journal=WAL&_timeout=5000")
	if err != nil {
		return db, err
	}

	return db, nil
}

func Close(db *sql.DB) {

	// https://www.sqlite.org/pragma.html#pragma_optimize
	// "Applications with short-lived database connections should run "PRAGMA optimize;" once, just prior to closing each database connection."

	sqlStatement := `PRAGMA optimize;`
	_, err := db.Query(sqlStatement)
	util.CheckAndPanic(err)

	db.Close()
}
