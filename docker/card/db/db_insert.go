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
		panic("expected one record to be inserted")
	}
}

func Db_insert_card_with_uid(key0 string, key1 string, k2 string, key3 string, key4 string,
	login string, password string, uid string, group_tag string) {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// insert a new card record
	sqlStatement := `INSERT INTO cards (key0_auth, key1_enc,` +
		` key2_cmac, key3, key4, login, password, uid, group_tag)` +
		` VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`
	res, err := db.Exec(sqlStatement, key0, key1, k2, key3, key4, login, password, uid, group_tag)
	util.Check(err)
	count, err := res.RowsAffected()
	util.Check(err)
	if count != 1 {
		panic("expected one record to be inserted")
	}
}

func Db_insert_program_cards(secret string,
	group_tag string, max_group_num int, initial_balance int,
	create_time int, expire_time int) {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// insert a new card record
	sqlStatement := `INSERT INTO program_cards (secret, group_tag,` +
		` max_group_num, initial_balance, create_time, expire_time)` +
		` VALUES ($1, $2, $3, $4, $5, $6);`
	res, err := db.Exec(sqlStatement, secret, group_tag, max_group_num, initial_balance, create_time, expire_time)
	util.Check(err)
	count, err := res.RowsAffected()
	util.Check(err)
	if count != 1 {
		panic("expected one record to be inserted")
	}
}
