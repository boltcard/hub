package db

import (
	"card/util"

	_ "github.com/mattn/go-sqlite3"
)

type CardKeys struct {
	Key0 string
	Key1 string
	Key2 string
	Key3 string
	Key4 string
}

func Db_wipe_card(card_id int) CardKeys {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// update card record
	sqlStatement := `UPDATE cards SET wiped = 'Y'` +
		` WHERE card_id = $6;`
	_, err = db.Exec(sqlStatement, card_id)
	util.Check(err)

	// get keys
	sqlStatement = `SELECT key0_auth, key1_enc, key2_cmac, key3, key4 FROM cards` +
		` WHERE card_id=$1;`
	row := db.QueryRow(sqlStatement, card_id)
	util.Check(err)

	var cardKeys CardKeys

	err = row.Scan(&cardKeys.Key0, &cardKeys.Key1, &cardKeys.Key2, &cardKeys.Key3, &cardKeys.Key4)
	util.Check(err)

	return cardKeys
}
