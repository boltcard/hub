package db

import (
	"card/util"

	_ "github.com/mattn/go-sqlite3"
)

func Db_update_tokens(initial_refresh_token string, new_refresh_token string, access_token string) (success bool) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE cards SET access_token = $1, refresh_token = $2` +
		` WHERE refresh_token = $3 AND wiped = 'N';`
	res, err := db.Exec(sqlStatement, access_token, new_refresh_token, initial_refresh_token)
	util.Check(err)
	count, err := res.RowsAffected()
	util.Check(err)

	return (count == 1)
}

func Db_update_card_with_pin(card_id int, tx_limit_sats int, day_limit_sats int, pin_enable string, pin_number string, pin_limit_sats int, lnurlw_enable string) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE cards SET tx_limit_sats = $1, day_limit_sats = $2, pin_enable = $3, pin_number = $4, pin_limit_sats = $5, lnurlw_enable = $6` +
		` WHERE card_id = $7 AND wiped = 'N';`
	_, err = db.Exec(sqlStatement, tx_limit_sats, day_limit_sats, pin_enable, pin_number, pin_limit_sats, lnurlw_enable, card_id)
	util.Check(err)
}

func Db_update_card_without_pin(card_id int, tx_limit_sats int, day_limit_sats int, pin_enable string, pin_limit_sats int, lnurlw_enable string) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE cards SET tx_limit_sats = $1, day_limit_sats = $2, pin_enable = $3, pin_limit_sats = $4, lnurlw_enable = $5` +
		` WHERE card_id = $6 AND wiped = 'N';`
	_, err = db.Exec(sqlStatement, tx_limit_sats, day_limit_sats, pin_enable, pin_limit_sats, lnurlw_enable, card_id)
	util.Check(err)
}

func Db_update_card_payment_fee(card_payment_id int, fee_sats int) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE card_payments SET fee_sats = $1 WHERE card_payment_id = $2;`
	_, err = db.Exec(sqlStatement, fee_sats, card_payment_id)
	util.Check(err)
}

func Db_update_card_payment_unpaid(card_payment_id int) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE card_payments SET paid = 'N' WHERE card_payment_id = $2;`
	_, err = db.Exec(sqlStatement, card_payment_id)
	util.Check(err)
}
