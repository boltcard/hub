package db

import (
	"card/util"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Db_add_card_receipt(db_conn *sql.DB, card_id int, payment_request string, payment_hash_hex string, amount_sats int) (card_receipt_id int) {

	// insert a new record
	sqlStatement := `INSERT INTO card_receipts (card_id, ln_invoice, r_hash_hex, amount_sats,` +
		` timestamp, expire_time)` +
		` VALUES ($1, $2, $3, $4, unixepoch(), unixepoch() + 86400);`
	res, err := db_conn.Exec(sqlStatement, card_id, payment_request, payment_hash_hex, amount_sats)
	util.CheckAndPanic(err)

	count, err := res.RowsAffected()
	util.CheckAndPanic(err)
	if count != 1 {
		panic("expected one record to be inserted")
	}

	card_receipt_id_int64, err := res.LastInsertId()
	util.CheckAndPanic(err)

	card_receipt_id = int(card_receipt_id_int64)
	return card_receipt_id
}

func Db_add_card_payment(db_conn *sql.DB, card_id int, amount_sat int, invoice string) (card_payment_id int) {

	// insert a new record
	sqlStatement := `INSERT INTO card_payments (card_id, amount_sats, ln_invoice,` +
		` timestamp, expire_time)` +
		` VALUES ($1, $2, $3, unixepoch(), unixepoch() + 86400);`
	res, err := db_conn.Exec(sqlStatement, card_id, amount_sat, invoice)
	util.CheckAndPanic(err)

	count, err := res.RowsAffected()
	util.CheckAndPanic(err)
	if count != 1 {
		panic("expected one record to be inserted")
	}

	card_payment_id_int64, err := res.LastInsertId()
	util.CheckAndPanic(err)

	card_payment_id = int(card_payment_id_int64)

	return card_payment_id
}
