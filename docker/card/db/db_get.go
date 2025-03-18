package db

import (
	"card/util"

	_ "github.com/mattn/go-sqlite3"
)

func Db_get_setting(name string) string {

	value := ""

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	sqlStatement := `select value from settings where name=$1;`

	row := db.QueryRow(sqlStatement, name)
	err = row.Scan(&value)
	if err != nil {
		return ""
	}

	return value
}

func Db_get_card_id_from_access_token(access_token string) (card_id int) {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	// get card id
	sqlStatement := `SELECT card_id FROM cards WHERE access_token=$1 AND wiped = 'N';`
	row := db.QueryRow(sqlStatement, access_token)
	util.Check(err)

	value := 0
	err = row.Scan(&value)
	if err != nil {
		return 0
	}

	return value
}

func Db_get_total_paid_receipts(card_id int) int {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	// get card id
	sqlStatement := `SELECT IFNULL(SUM(amount_sats),0) FROM card_receipts` +
		` WHERE paid_flag='Y' AND card_id=$1;`
	row := db.QueryRow(sqlStatement, card_id)
	util.Check(err)

	value := 0
	err = row.Scan(&value)
	if err != nil {
		return 0
	}

	return value
}

func Db_get_total_paid_payments(card_id int) int {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	// get card id
	sqlStatement := `SELECT IFNULL(SUM(amount_sats) + SUM(fee_sats),0) FROM card_payments` +
		` WHERE paid_flag='Y' AND card_id=$1;`
	row := db.QueryRow(sqlStatement, card_id)
	util.Check(err)

	value := 0
	err = row.Scan(&value)
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

func Db_get_card_keys() CardLookups {
	var cardLookups CardLookups

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	// get card id
	sqlStatement := `SELECT card_id,` +
		` key1_enc, key2_cmac,` +
		` uid` +
		` FROM cards` +
		` WHERE wiped = 'N';`
	rows, err := db.Query(sqlStatement)
	util.Check(err)

	for rows.Next() {
		var cardLookup CardLookup

		err := rows.Scan(
			&cardLookup.CardId,
			&cardLookup.Key1,
			&cardLookup.Key2,
			&cardLookup.UID)
		util.Check(err)

		cardLookups = append(cardLookups, cardLookup)
	}

	return cardLookups
}

func Db_get_card_counter(cardId int) (counter uint32) {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	sqlStatement := `SELECT last_counter_value FROM cards` +
		` WHERE card_id=$1 AND wiped = 'N';`
	row := db.QueryRow(sqlStatement, cardId)
	util.Check(err)

	var value uint32 = 0
	err = row.Scan(&value)
	if err != nil {
		return 0
	}

	return value
}

func Db_get_lnurlw_k1(lnurlw_k1 string) (card_id int, lnurlw_k1_expiry uint64) {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	sqlStatement := `SELECT card_id, lnurlw_k1_expiry FROM cards` +
		` WHERE lnurlw_k1=$1 AND wiped = 'N';`
	row := db.QueryRow(sqlStatement, lnurlw_k1)
	util.Check(err)

	card_id = 0
	lnurlw_k1_expiry = 0
	err = row.Scan(&card_id, &lnurlw_k1_expiry)
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

func Db_get_card(card_id int) (card *Card, err error) {
	c := Card{}

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer Close(db)

	sqlStatement := `SELECT card_id, key0_auth, key1_enc, ` +
		`key2_cmac, key3, key4, login, password, access_token, ` +
		`refresh_token, uid, last_counter_value, ` +
		`lnurlw_request_timeout_sec, lnurlw_enable, ` +
		`lnurlw_k1, lnurlw_k1_expiry, tx_limit_sats, ` +
		`day_limit_sats, uid_privacy, pin_enable, pin_number, ` +
		`pin_limit_sats, wiped FROM cards WHERE card_id=$1 AND wiped = 'N';`
	row := db.QueryRow(sqlStatement, card_id)
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
