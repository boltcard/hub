package db

import (
	"database/sql"
	"errors"

	log "github.com/sirupsen/logrus"
)

func Db_set_setting(db_conn *sql.DB, name string, value string) {

	// ensure no records with the same name exist
	sqlStatement := `DELETE FROM settings` +
		` WHERE name = $1;`
	_, err := db_conn.Exec(sqlStatement, name)
	if err != nil {
		log.Error("db_set_setting delete error: ", err)
		return
	}

	// insert a new record into settings table
	sqlStatement = `INSERT INTO settings` +
		` (name, value)` +
		` VALUES ($1, $2);`
	res, err := db_conn.Exec(sqlStatement, name, value)
	if err != nil {
		log.Error("db_set_setting insert error: ", err)
		return
	}
	count, err := res.RowsAffected()
	if err != nil {
		log.Error("db_set_setting rows affected error: ", err)
		return
	}
	if count != 1 {
		log.Error("db_set_setting: expected one setting record to be inserted")
	}
}

func Db_set_tokens(db_conn *sql.DB, login string, password string,
	access_token string, refresh_token string) error {

	// update card record
	sqlStatement := `UPDATE cards` +
		` SET access_token = $1, refresh_token = $2` +
		` WHERE login = $3 AND password = $4 AND wiped = 'N';`
	res, err := db_conn.Exec(sqlStatement, access_token, refresh_token, login, password)
	if err != nil {
		log.Error("db_set_tokens exec error: ", err)
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		log.Error("db_set_tokens rows affected error: ", err)
		return err
	}

	if count != 1 {
		return errors.New("login not valid")
	}

	return nil
}

func Db_set_receipt_paid(db_conn *sql.DB, paymentHash string) {

	// update card record
	sqlStatement := `UPDATE card_receipts SET paid_flag = 'Y'` +
		` WHERE r_hash_hex = $1;`
	_, err := db_conn.Exec(sqlStatement, paymentHash)
	if err != nil {
		log.Error("db_set_receipt_paid error: ", err)
	}
}

func Db_set_card_keys(db_conn *sql.DB, card_id int, key0 string, key1 string, k2 string, key3 string, key4 string) {

	// update card record
	sqlStatement := `UPDATE cards SET key0_auth = $1, key1_enc = $2,` +
		` key2_cmac = $3, key3 = $4, key4 = $5` +
		` WHERE card_id = $6 AND wiped = 'N';`
	_, err := db_conn.Exec(sqlStatement, key0, key1, k2, key3, key4, card_id)
	if err != nil {
		log.Error("db_set_card_keys error: ", err)
	}
}

func Db_set_card_counter(db_conn *sql.DB, cardId int, counter_value uint32) {

	// update card record
	sqlStatement := `UPDATE cards SET last_counter_value = $1` +
		` WHERE card_id = $2 AND wiped = 'N';`
	_, err := db_conn.Exec(sqlStatement, counter_value, cardId)
	if err != nil {
		log.Error("db_set_card_counter error: ", err)
	}
}

func Db_set_lnurlw_k1(db_conn *sql.DB, cardId int, lnurlwK1 string, lnurlwK1Expiry int64) {

	// update card record
	sqlStatement := `UPDATE cards SET lnurlw_k1 = $1, lnurlw_k1_expiry = $2` +
		` WHERE card_id = $3 AND wiped = 'N';`
	_, err := db_conn.Exec(sqlStatement, lnurlwK1, lnurlwK1Expiry, cardId)
	if err != nil {
		log.Error("db_set_lnurlw_k1 error: ", err)
	}
}
