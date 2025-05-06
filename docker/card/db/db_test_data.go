package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func add_test_data(db_conn *sql.DB) {

	// can set up some test data here

	// new test values for
	// - invite_secret
	// - key0, key1, key2, key3, key4
	// - login, password
	// can each be generated as a random
	// 128 bit hex encoded string like this ..
	//
	// $ hexdump -vn16 -e'4/4 "%08x" 1 "\n"' /dev/random

	// invite_secret must not be easy to brute force

	// Db_set_setting("invite_secret", "random-128-bit-as-hex-encoded-string")

	// add a test card

	// Db_insert_card(
	// 	"random-128-bit-as-hex-encoded-string", "random-128-bit-as-hex-encoded-string",
	// 	"random-128-bit-as-hex-encoded-string", "random-128-bit-as-hex-encoded-string",
	// 	"random-128-bit-as-hex-encoded-string", "random-128-bit-as-hex-encoded-string",
	// 	"random-128-bit-as-hex-encoded-string")

	// add a receipt record to give the card a small balance
	// fake payment request & payment hash here are safe
	// they were copied from the example on this public page
	// https://phoenix.acinq.co/server/api

	// Db_add_card_receipt(
	// 	1,
	// 	"lntb1u1pjlsjnqpp57svjqly7mh5sy84lk67sma4apfnqdm90jdf40np0xchrpq6uxajscqpjsp592kp0fs2ssgpq9h54tsfaj5w34287v8fezgaw6cr56f076c05glq9q7sqqqqqqqqqqqqqqqqqqqsqqqqqysgqdq6d4ujqenfwfehggrfdemx76trv5mqz9grzjqwfn3p9278ttzzpe0e00uhyxhned3j5d9acqak5emwfpflp8z2cnflcyamh4dcuhwqqqqqlgqqqqqeqqjqnjpjvnv0p3wvwc6vhzkkgm8kl9r837x4p9qupk5ln5tqlm7prrlsy5xd8cf5agae64f53dvm9el0z5hvgcnta4stgmrg7zwfah0nqrqph4ts8l",
	// 	"f419207c9edde9021ebfb6bd0df6bd0a6606ecaf935357cc2f362e30835c3765",
	// 	1000)
	// Db_set_receipt_paid("f419207c9edde9021ebfb6bd0df6bd0a6606ecaf935357cc2f362e30835c3765")
}
