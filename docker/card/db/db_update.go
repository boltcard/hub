package db

import (
	"database/sql"

	log "github.com/sirupsen/logrus"
)

func Db_update_tokens(db_conn *sql.DB, initial_refresh_token string, new_refresh_token string, access_token string) (success bool) {

	// update record
	sqlStatement := `UPDATE cards SET access_token = $1, refresh_token = $2` +
		` WHERE refresh_token = $3 AND wiped = 'N';`
	res, err := db_conn.Exec(sqlStatement, access_token, new_refresh_token, initial_refresh_token)
	if err != nil {
		log.Error("db_update_tokens exec error: ", err)
		return false
	}
	count, err := res.RowsAffected()
	if err != nil {
		log.Error("db_update_tokens rows affected error: ", err)
		return false
	}

	return (count == 1)
}

func Db_update_card_with_pin(db_conn *sql.DB, card_id int, tx_limit_sats int, day_limit_sats int, pin_enable string, pin_number string, pin_limit_sats int, lnurlw_enable string) {

	// update record
	sqlStatement := `UPDATE cards SET tx_limit_sats = $1, day_limit_sats = $2, pin_enable = $3, pin_number = $4, pin_limit_sats = $5, lnurlw_enable = $6` +
		` WHERE card_id = $7 AND wiped = 'N';`
	_, err := db_conn.Exec(sqlStatement, tx_limit_sats, day_limit_sats, pin_enable, pin_number, pin_limit_sats, lnurlw_enable, card_id)
	if err != nil {
		log.Error("db_update_card_with_pin error: ", err)
	}
}

func Db_update_card_without_pin(db_conn *sql.DB, card_id int, tx_limit_sats int, day_limit_sats int, pin_enable string, pin_limit_sats int, lnurlw_enable string) {

	// update record
	sqlStatement := `UPDATE cards SET tx_limit_sats = $1, day_limit_sats = $2, pin_enable = $3, pin_limit_sats = $4, lnurlw_enable = $5` +
		` WHERE card_id = $6 AND wiped = 'N';`
	_, err := db_conn.Exec(sqlStatement, tx_limit_sats, day_limit_sats, pin_enable, pin_limit_sats, lnurlw_enable, card_id)
	if err != nil {
		log.Error("db_update_card_without_pin error: ", err)
	}
}

func Db_update_card_payment_fee(db_conn *sql.DB, card_payment_id int, fee_sats int) {

	// update record
	sqlStatement := `UPDATE card_payments SET fee_sats = $1 WHERE card_payment_id = $2;`
	_, err := db_conn.Exec(sqlStatement, fee_sats, card_payment_id)
	if err != nil {
		log.Error("db_update_card_payment_fee error: ", err)
	}
}

func Db_update_card_payment_unpaid(db_conn *sql.DB, card_payment_id int) {

	// update record
	sqlStatement := `UPDATE card_payments SET paid_flag = 'N' WHERE card_payment_id = $1;`
	_, err := db_conn.Exec(sqlStatement, card_payment_id)
	if err != nil {
		log.Error("db_update_card_payment_unpaid error: ", err)
	}
}

func Db_update_card_note(db_conn *sql.DB, card_id int, note string) {

	// update record
	sqlStatement := `UPDATE cards SET note = $1 WHERE card_id = $2 AND wiped = 'N';`
	_, err := db_conn.Exec(sqlStatement, note, card_id)
	if err != nil {
		log.Error("db_update_card_note error: ", err)
	}
}

func Db_update_receipt_paid(db_conn *sql.DB, card_receipt_id int) {

	// update record
	sqlStatement := `UPDATE card_receipts SET paid_flag = 'Y'` +
		` WHERE card_receipt_id = $1;`
	_, err := db_conn.Exec(sqlStatement, card_receipt_id)
	if err != nil {
		log.Error("db_update_receipt_paid error: ", err)
	}
}
