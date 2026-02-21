package db

import (
	"card/util"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Db_get_setting(db_conn *sql.DB, name string) string {

	value := ""

	sqlStatement := `select value from settings where name=$1;`

	row := db_conn.QueryRow(sqlStatement, name)
	err := row.Scan(&value)
	if err != nil {
		return ""
	}

	return value
}

func Db_get_card_count(db_conn *sql.DB) (count int, err error) {

	sqlStatement := `select count(*) from cards;`

	row := db_conn.QueryRow(sqlStatement)
	err = row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return
}

func Db_get_card_id_from_access_token(db_conn *sql.DB, access_token string) (card_id int) {

	// get card id
	sqlStatement := `SELECT card_id FROM cards WHERE access_token=$1 AND wiped = 'N';`
	row := db_conn.QueryRow(sqlStatement, access_token)

	value := 0
	err := row.Scan(&value)
	if err != nil {
		return 0
	}

	return value
}

func Db_get_total_paid_receipts(db_conn *sql.DB, card_id int) int {

	// get card id
	sqlStatement := `SELECT IFNULL(SUM(amount_sats),0) FROM card_receipts` +
		` WHERE paid_flag='Y' AND card_id=$1;`
	row := db_conn.QueryRow(sqlStatement, card_id)

	value := 0
	err := row.Scan(&value)
	if err != nil {
		return 0
	}

	return value
}

func Db_get_card_balance(db_conn *sql.DB, card_id int) int {
	sqlStatement := `SELECT
		IFNULL((SELECT SUM(amount_sats) FROM card_receipts WHERE paid_flag='Y' AND card_id=$1), 0) -
		IFNULL((SELECT SUM(amount_sats) + SUM(fee_sats) FROM card_payments WHERE paid_flag='Y' AND card_id=$1), 0)`
	row := db_conn.QueryRow(sqlStatement, card_id)
	value := 0
	err := row.Scan(&value)
	if err != nil {
		return 0
	}
	return value
}

type CardLookup struct {
	CardId int
	Key1   string
	Key2   string
	UID    string
}

type CardLookups []CardLookup

func Db_get_card_keys(db_conn *sql.DB) CardLookups {

	var cardLookups CardLookups

	// get card id
	sqlStatement := `SELECT card_id,` +
		` key1_enc, key2_cmac,` +
		` uid` +
		` FROM cards` +
		` WHERE wiped = 'N';`
	rows, err := db_conn.Query(sqlStatement)
	util.CheckAndPanic(err)
	defer rows.Close()

	for rows.Next() {
		var cardLookup CardLookup

		err := rows.Scan(
			&cardLookup.CardId,
			&cardLookup.Key1,
			&cardLookup.Key2,
			&cardLookup.UID)
		util.CheckAndPanic(err)

		cardLookups = append(cardLookups, cardLookup)
	}

	return cardLookups
}

func Db_get_card_counter(db_conn *sql.DB, cardId int) (counter uint32) {

	sqlStatement := `SELECT last_counter_value FROM cards` +
		` WHERE card_id=$1 AND wiped = 'N';`
	row := db_conn.QueryRow(sqlStatement, cardId)

	var value uint32 = 0
	err := row.Scan(&value)
	if err != nil {
		return 0
	}

	return value
}

func Db_get_lnurlw_k1(db_conn *sql.DB, lnurlw_k1 string) (card_id int, lnurlw_k1_expiry uint64) {

	sqlStatement := `SELECT card_id, lnurlw_k1_expiry FROM cards` +
		` WHERE lnurlw_k1=$1 AND wiped = 'N';`
	row := db_conn.QueryRow(sqlStatement, lnurlw_k1)

	card_id = 0
	lnurlw_k1_expiry = 0
	err := row.Scan(&card_id, &lnurlw_k1_expiry)
	if err != nil {
		return 0, 0
	}

	return card_id, lnurlw_k1_expiry
}

type Card struct {
	Card_id                    int
	Key0_auth                  string
	Key1_enc                   string
	Key2_cmac                  string
	Key3                       string
	Key4                       string
	Login                      string
	Password                   string
	Access_token               string
	Refresh_token              string
	Uid                        string
	Last_counter_value         int
	Lnurlw_request_timeout_sec int
	Lnurlw_enable              string
	Lnurlw_k1                  string
	Lnurlw_k1_expiry           int
	Tx_limit_sats              int
	Day_limit_sats             int
	Uid_privacy                string
	Pin_enable                 string
	Pin_number                 string
	Pin_limit_sats             int
	Wiped                      string
}

func Db_get_card(db_conn *sql.DB, card_id int) (card *Card, err error) {

	c := Card{}

	sqlStatement := `SELECT card_id, key0_auth, key1_enc, ` +
		`key2_cmac, key3, key4, login, password, access_token, ` +
		`refresh_token, uid, last_counter_value, ` +
		`lnurlw_request_timeout_sec, lnurlw_enable, ` +
		`lnurlw_k1, lnurlw_k1_expiry, tx_limit_sats, ` +
		`day_limit_sats, uid_privacy, pin_enable, pin_number, ` +
		`pin_limit_sats, wiped FROM cards WHERE card_id=$1 AND wiped = 'N';`
	row := db_conn.QueryRow(sqlStatement, card_id)
	err = row.Scan(
		&c.Card_id,
		&c.Key0_auth,
		&c.Key1_enc,
		&c.Key2_cmac,
		&c.Key3,
		&c.Key4,
		&c.Login,
		&c.Password,
		&c.Access_token,
		&c.Refresh_token,
		&c.Uid,
		&c.Last_counter_value,
		&c.Lnurlw_request_timeout_sec,
		&c.Lnurlw_enable,
		&c.Lnurlw_k1,
		&c.Lnurlw_k1_expiry,
		&c.Tx_limit_sats,
		&c.Day_limit_sats,
		&c.Uid_privacy,
		&c.Pin_enable,
		&c.Pin_number,
		&c.Pin_limit_sats,
		&c.Wiped)

	return &c, err
}

