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

	/*
		stats := db.Stats()
		log.Info(
			"db Idle=" + strconv.Itoa(stats.Idle) +
				", InUse=" + strconv.Itoa(stats.InUse) +
				", WaitCount=" + strconv.Itoa(int(stats.WaitCount)) +
				", WaitDuration (ms)=" + strconv.Itoa(int(stats.WaitDuration.Milliseconds())))

		// https://www.alexedwards.net/blog/configuring-sqldb
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(1 * time.Hour)
	*/

	return db, nil
}
