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

	// update schema for new versions

	// set initial data
	if Db_get_setting("host_domain") == "" {

		// compatible with current (June 2024) BoltCardWallet app
		Db_set_setting("invite_secret", "")

		hostDomain := os.Getenv("HOST_DOMAIN")
		Db_set_setting("host_domain", hostDomain)

		gcUrl := os.Getenv("GC_URL")
		Db_set_setting("gc_url", gcUrl)

		Db_set_setting("log_level", "debug")
		Db_set_setting("min_withdraw_sats", "1")
		Db_set_setting("max_withdraw_sats", "100000000")

		// set password salt
		passwordSalt := util.Random_hex()
		Db_set_setting("admin_password_salt", passwordSalt)

		add_test_data()
	}
}
