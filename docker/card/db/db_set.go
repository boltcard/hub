package db

import (
	"card/util"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

func Db_set_setting(name string, value string) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// ensure no records with the same name exist
	sqlStatement := `DELETE FROM settings` +
		` WHERE name = $1;`
	_, err = db.Exec(sqlStatement, name)
	util.Check(err)

	// insert a new record into settings table
	sqlStatement = `INSERT INTO settings` +
		` (name, value)` +
		` VALUES ($1, $2);`
	res, err := db.Exec(sqlStatement, name, value)
	util.Check(err)
	count, err := res.RowsAffected()
	util.Check(err)
	if count != 1 {
		panic("expected one setting record to be inserted")
	}
}

func Db_set_tokens(login string, password string,
	access_token string, refresh_token string) error {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE cards` +
		` SET access_token = $1, refresh_token = $2` +
		` WHERE login = $3 AND password = $4 AND wiped = 'N';`
	res, err := db.Exec(sqlStatement, access_token, refresh_token, login, password)
	util.Check(err)
	count, err := res.RowsAffected()
	util.Check(err)

	if count != 1 {
		return errors.New("login not valid")
	}

	return nil
}

func Db_set_receipt_paid(paymentHash string) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE card_receipts SET paid_flag = 'Y'` +
		` WHERE r_hash_hex = $1;`
	_, err = db.Exec(sqlStatement, paymentHash)
	util.Check(err)
}

func Db_set_card_keys(card_id int, key0 string, key1 string, k2 string, key3 string, key4 string) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE cards SET key0_auth = $1, key1_enc = $2,` +
		` key2_cmac = $3, key3 = $4, key4 = $5` +
		` WHERE card_id = $6 AND wiped = 'N';`
	_, err = db.Exec(sqlStatement, key0, key1, k2, key3, key4, card_id)
	util.Check(err)
}

func Db_set_card_counter(cardId int, counter_value uint32) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE cards SET last_counter_value = $1` +
		` WHERE card_id = $2 AND wiped = 'N';`
	_, err = db.Exec(sqlStatement, counter_value, cardId)
	util.Check(err)
}

func Db_set_lnurlw_k1(cardId int, lnurlwK1 string, lnurlwK1Expiry int64) {

	// open a database connection
	db, err := Open()
	util.Check(err)

	// update card record
	sqlStatement := `UPDATE cards SET lnurlw_k1 = $1, lnurlw_k1_expiry = $2` +
		` WHERE card_id = $3 AND wiped = 'N';`
	_, err = db.Exec(sqlStatement, lnurlwK1, lnurlwK1Expiry, cardId)
	util.Check(err)
}
