package db

import (
	"card/util"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func Db_init() {

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// ensure tables exist (idempotent)
	create_settings_table(db)
	create_cards_table(db)
	create_card_payments_table(db)
	create_card_receipts_table(db)

	// update schema and track with a 'schema_version_number' setting
	if Db_get_setting("schema_version_number") == "" {
		Db_set_setting("schema_version_number", "1")
	}

	if Db_get_setting("schema_version_number") == "1" {
		update_schema_1(db) // fee_sats columns
	}

	if Db_get_setting("schema_version_number") == "2" {
		update_schema_2(db) // group_tag column
	}

	if Db_get_setting("schema_version_number") == "3" {
		update_schema_3(db) // program_cards table
	}

	if Db_get_setting("schema_version_number") != "4" {
		panic("database schema is not as expected")
	}

	// set initial data
	if Db_get_setting("host_domain") == "" {

		// compatible with current (June 2024) BoltCardWallet app
		Db_set_setting("invite_secret", "")

		hostDomain := os.Getenv("HOST_DOMAIN")
		Db_set_setting("host_domain", hostDomain)
		Db_set_setting("gc_url", "")
		Db_set_setting("log_level", "debug")
		Db_set_setting("min_withdraw_sats", "1")
		Db_set_setting("max_withdraw_sats", "100000000")

		// set password salt
		passwordSalt := util.Random_hex()
		Db_set_setting("admin_password_salt", passwordSalt)

		// set new card code
		newCardCode := util.Random_hex()
		Db_set_setting("new_card_code", newCardCode)

		add_test_data()
	}
}
