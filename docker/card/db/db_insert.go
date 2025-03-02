package db

import (
	"card/util"

	_ "github.com/mattn/go-sqlite3"
)

func Db_insert_card(key0 string, key1 string, k2 string, key3 string, key4 string,
	login string, password string) {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// insert a new card record
	sqlStatement := `INSERT INTO cards (key0_auth, key1_enc,` +
		` key2_cmac, key3, key4, login, password)` +
		` VALUES ($1, $2, $3, $4, $5, $6, $7);`
	res, err := db.Exec(sqlStatement, key0, key1, k2, key3, key4, login, password)
	util.Check(err)
	count, err := res.RowsAffected()
	util.Check(err)
	if count != 1 {
		panic("expected one setting record to be inserted")
	}
}
