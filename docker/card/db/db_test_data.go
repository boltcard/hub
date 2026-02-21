package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func add_test_data(db_conn *sql.DB) {
	// placeholder for test data setup during development
}
