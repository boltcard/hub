package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func create_settings_table(db *sql.DB) {

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS
		settings (
			setting_id INTEGER PRIMARY KEY,
			name VARCHAR(30) UNIQUE NOT NULL DEFAULT '',
			value VARCHAR(128) NOT NULL DEFAULT ''
		);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}

func create_cards_table(db *sql.DB) {

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS
		cards (
			card_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			key0_auth CHAR(32) NOT NULL,
			key1_enc CHAR(32) NOT NULL,
			key2_cmac CHAR(32) NOT NULL,
			key3 CHAR(32) NOT NULL,
			key4 CHAR(32) NOT NULL,
			login CHAR(32) NOT NULL,
			password CHAR(32) NOT NULL,
			access_token CHAR(32) NOT NULL DEFAULT '',
			refresh_token CHAR(32) NOT NULL DEFAULT '',
			uid VARCHAR(14) NOT NULL DEFAULT '',
			last_counter_value INT NOT NULL DEFAULT 0,
			lnurlw_request_timeout_sec INT NOT NULL DEFAULT 10,
			lnurlw_enable CHAR(1) NOT NULL DEFAULT 'N',
			lnurlw_k1 CHAR(32) NOT NULL DEFAULT '',
			lnurlw_k1_expiry INT NOT NULL DEFAULT 0,
			tx_limit_sats INT NOT NULL DEFAULT 1000000,
			day_limit_sats INT NOT NULL DEFAULT 0,
			uid_privacy CHAR(1) NOT NULL DEFAULT 'N',
			pin_enable CHAR(1) NOT NULL DEFAULT 'N',
			pin_number CHAR(4) NOT NULL DEFAULT '0000',
			pin_limit_sats INT NOT NULL DEFAULT 0,
			wiped CHAR(1) NOT NULL DEFAULT 'N'
		);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}

func create_card_payments_table(db *sql.DB) {

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS
		card_payments (
			card_payment_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			card_id INT NOT NULL,
			ln_invoice VARCHAR(1024) NOT NULL DEFAULT '',
			amount_sats INTEGER NOT NULL DEFAULT 0,
			paid_flag CHAR(1) NOT NULL DEFAULT 'Y',
			timestamp INTEGER NOT NULL,
			expire_time INTEGER NOT NULL,
			FOREIGN KEY(card_id) REFERENCES cards(card_id)
		);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}

func create_card_receipts_table(db *sql.DB) {

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS
		card_receipts (
			card_receipt_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			card_id INTEGER NOT NULL,
			ln_invoice VARCHAR(1024) NOT NULL DEFAULT '',
			r_hash_hex CHAR(64) UNIQUE NOT NULL DEFAULT '',
			amount_sats INTEGER CHECK (amount_sats > 0),
			paid_flag CHAR(1) NOT NULL DEFAULT 'N',
			timestamp INTEGER NOT NULL,
			expire_time INTEGER NOT NULL,
			CONSTRAINT fk_card FOREIGN KEY(card_id) REFERENCES cards(card_id)
		);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}

func update_schema_1(db *sql.DB) {

	sqlStmt := `
		BEGIN TRANSACTION;
		ALTER TABLE card_payments ADD COLUMN fee_sats INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE card_receipts ADD COLUMN fee_sats INTEGER NOT NULL DEFAULT 0;
		UPDATE settings SET value='2' WHERE name='schema_version_number';
		COMMIT TRANSACTION;
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}

func update_schema_2(db *sql.DB) {

	sqlStmt := `
		BEGIN TRANSACTION;
		ALTER TABLE cards ADD COLUMN group_tag TEXT NOT NULL DEFAULT '';
		UPDATE settings SET value='3' WHERE name='schema_version_number';
		COMMIT TRANSACTION;
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}

func update_schema_3(db *sql.DB) {

	sqlStmt := `
		BEGIN TRANSACTION;
		CREATE TABLE IF NOT EXISTS
		program_cards (
			program_card_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			secret TEXT NOT NULL DEFAULT '',
			group_tag TEXT NOT NULL DEFAULT '',
			max_group_num INTEGER NOT NULL DEFAULT 0,
			initial_balance INTEGER NOT NULL DEFAULT 0,
			create_time INTEGER NOT NULL,
			expire_time INTEGER NOT NULL
		);
		UPDATE settings SET value='4' WHERE name='schema_version_number';
		COMMIT TRANSACTION;
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}

func update_schema_4(db *sql.DB) {

	sqlStmt := `
		BEGIN TRANSACTION;
		CREATE INDEX IF NOT EXISTS idx_cards_uid ON cards(uid);
		CREATE INDEX IF NOT EXISTS idx_cards_group_tag ON cards(group_tag);
		CREATE INDEX IF NOT EXISTS idx_card_payments_card_id ON card_payments(card_id);
		CREATE INDEX IF NOT EXISTS idx_card_receipts_card_id ON card_receipts(card_id);
		CREATE INDEX IF NOT EXISTS idx_program_cards_group_tag ON program_cards(group_tag);
		CREATE INDEX IF NOT EXISTS idx_card_payments_timestamp ON card_payments(timestamp);
		CREATE INDEX IF NOT EXISTS idx_card_receipts_timestamp ON card_receipts(timestamp);
		UPDATE settings SET value='5' WHERE name='schema_version_number';
		COMMIT TRANSACTION;
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q : %s\n", err, sqlStmt)
		return
	}
}
