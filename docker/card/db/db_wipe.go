package db

import (
	"card/util"
	"database/sql"
)

type CardKeys struct {
	Key0 string
	Key1 string
	Key2 string
	Key3 string
	Key4 string
}

func Db_wipe_card(db_conn *sql.DB, card_id int) CardKeys {

	// update card record
	sqlStatement := `UPDATE cards SET wiped = 'Y'` +
		` WHERE card_id = $1;`
	_, err := db_conn.Exec(sqlStatement, card_id)
	util.CheckAndPanic(err)

	// get keys
	sqlStatement = `SELECT key0_auth, key1_enc, key2_cmac, key3, key4 FROM cards` +
		` WHERE card_id=$1;`
	row := db_conn.QueryRow(sqlStatement, card_id)
	util.CheckAndPanic(err)

	var cardKeys CardKeys

	err = row.Scan(&cardKeys.Key0, &cardKeys.Key1, &cardKeys.Key2, &cardKeys.Key3, &cardKeys.Key4)
	util.CheckAndPanic(err)

	return cardKeys
}
