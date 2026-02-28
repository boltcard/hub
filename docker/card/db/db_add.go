package db

import (
	"database/sql"

	log "github.com/sirupsen/logrus"
)

func Db_add_card_receipt(db_conn *sql.DB, card_id int, payment_request string, payment_hash_hex string, amount_sats int) (card_receipt_id int) {

	// insert a new record
	sqlStatement := `INSERT INTO card_receipts (card_id, ln_invoice, r_hash_hex, amount_sats,` +
		` timestamp, expire_time)` +
		` VALUES ($1, $2, $3, $4, unixepoch(), unixepoch() + 86400);`
	res, err := db_conn.Exec(sqlStatement, card_id, payment_request, payment_hash_hex, amount_sats)
	if err != nil {
		log.Error("db_add_card_receipt exec error: ", err)
		return 0
	}

	count, err := res.RowsAffected()
	if err != nil {
		log.Error("db_add_card_receipt rows affected error: ", err)
		return 0
	}
	if count != 1 {
		log.Error("db_add_card_receipt: expected one record to be inserted")
		return 0
	}

	card_receipt_id_int64, err := res.LastInsertId()
	if err != nil {
		log.Error("db_add_card_receipt last insert id error: ", err)
		return 0
	}

	card_receipt_id = int(card_receipt_id_int64)
	return card_receipt_id
}

func Db_add_card_payment(db_conn *sql.DB, card_id int, amount_sat int, invoice string) (card_payment_id int) {

	// insert a new record
	sqlStatement := `INSERT INTO card_payments (card_id, amount_sats, ln_invoice,` +
		` timestamp, expire_time)` +
		` VALUES ($1, $2, $3, unixepoch(), unixepoch() + 86400);`
	res, err := db_conn.Exec(sqlStatement, card_id, amount_sat, invoice)
	if err != nil {
		log.Error("db_add_card_payment exec error: ", err)
		return 0
	}

	count, err := res.RowsAffected()
	if err != nil {
		log.Error("db_add_card_payment rows affected error: ", err)
		return 0
	}
	if count != 1 {
		log.Error("db_add_card_payment: expected one record to be inserted")
		return 0
	}

	card_payment_id_int64, err := res.LastInsertId()
	if err != nil {
		log.Error("db_add_card_payment last insert id error: ", err)
		return 0
	}

	card_payment_id = int(card_payment_id_int64)

	return card_payment_id
}
