package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Open() (*sql.DB, error) {

	// https://github.com/mattn/go-sqlite3
	// section for..
	// Error: database is locked

	// busy timeout is in ms
	db, err := sql.Open("sqlite3", "/card_data/cards.db?cache=shared&mode=rwc&_busy_timeout=1000")
	if err != nil {
		return db, err
	}

	db.SetMaxOpenConns(1)

	return db, nil
}
