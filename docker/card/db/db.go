package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Open() (*sql.DB, error) {

	db, err := sql.Open("sqlite3", "/card_data/cards.db")
	if err != nil {
		return db, err
	}

	return db, nil
}
