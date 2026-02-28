package db

import (
	"card/util"
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func Db_init(db_conn *sql.DB) {

	// ensure tables exist (idempotent)
	create_settings_table(db_conn)
	create_cards_table(db_conn)
	create_card_payments_table(db_conn)
	create_card_receipts_table(db_conn)

	// update schema and track with a 'schema_version_number' setting
	if Db_get_setting(db_conn, "schema_version_number") == "" {
		Db_set_setting(db_conn, "schema_version_number", "1")
	}

	if Db_get_setting(db_conn, "schema_version_number") == "1" {
		update_schema_1(db_conn) // fee_sats columns
	}

	if Db_get_setting(db_conn, "schema_version_number") == "2" {
		update_schema_2(db_conn) // group_tag column
	}

	if Db_get_setting(db_conn, "schema_version_number") == "3" {
		update_schema_3(db_conn) // program_cards table
	}

	if Db_get_setting(db_conn, "schema_version_number") == "4" {
		update_schema_4(db_conn) // indexes
	}

	if Db_get_setting(db_conn, "schema_version_number") == "5" {
		update_schema_5(db_conn) // note column
	}

	if Db_get_setting(db_conn, "schema_version_number") != "6" {
		panic("database schema is not as expected")
	}

	// set initial data
	if Db_get_setting(db_conn, "host_domain") == "" {

		// compatible with current (June 2024) BoltCardWallet app
		Db_set_setting(db_conn, "invite_secret", "")

		hostDomain := os.Getenv("HOST_DOMAIN")
		if hostDomain == "" {
			panic("HOST_DOMAIN environment variable must be set")
		}
		Db_set_setting(db_conn, "host_domain", hostDomain)
		Db_set_setting(db_conn, "log_level", "info")

		// set password salt
		passwordSalt := util.Random_hex()
		Db_set_setting(db_conn, "admin_password_salt", passwordSalt)

		// set new card code
		newCardCode := util.Random_hex()
		Db_set_setting(db_conn, "new_card_code", newCardCode)

		add_test_data(db_conn)
	}
}
