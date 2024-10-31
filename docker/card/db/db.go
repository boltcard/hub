package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Open() (*sql.DB, error) {

	// https://github.com/mattn/go-sqlite3
	// section for..
	// Error: database is locked

	// https://phiresky.github.io/blog/2020/sqlite-performance-tuning/

	// busy timeout is in ms
	db, err := sql.Open("sqlite3", "/card_data/cards.db?_journal=WAL&_timeout=5000")
	if err != nil {
		return db, err
	}

	db.SetMaxOpenConns(1)

	return db, nil
}
