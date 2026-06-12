package main

import (
	"card/db"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openCliTestDB(t *testing.T) *sql.DB {
	t.Helper()
	os.Setenv("HOST_DOMAIN", "test.example.com")
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	db.Db_init(conn)
	return conn
}

func TestGetBalance_SumsTxs(t *testing.T) {
	conn := openCliTestDB(t)
	db.Db_insert_card(conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass")

	// paid receipt of 1000, paid payment of 200 (+10 fee)
	db.Db_add_card_receipt(conn, 1, "inv", "h1", 1000)
	db.Db_set_receipt_paid(conn, "h1", "test")
	payId := db.Db_add_card_payment(conn, 1, 200, "p1")
	db.Db_update_card_payment_fee(conn, payId, 10)

	// getBalance sums signed amounts from Db_select_card_txs:
	// receipt +1000, payment -(200) -(10) = 790
	if bal := getBalance(conn, 1); bal != 790 {
		t.Fatalf("expected balance 790, got %d", bal)
	}
}

func TestSetupCardAmountForTag_LoadsCards(t *testing.T) {
	conn := openCliTestDB(t)
	db.Db_insert_card_with_uid(conn, "k0", "k1", "k2", "k3", "k4", "l1", "p1", "uid1", "event1")
	db.Db_insert_card_with_uid(conn, "k0", "k1", "k2", "k3", "k4", "l2", "p2", "uid2", "event1")

	setupCardAmountForTag(conn, []string{"SetupCardAmountForTag", "event1", "5000"})

	for _, cardId := range []int{1, 2} {
		if bal := getBalance(conn, cardId); bal != 5000 {
			t.Fatalf("card %d: expected balance 5000, got %d", cardId, bal)
		}
	}
}

func TestSetupCardAmountForTag_MissingArgs(t *testing.T) {
	conn := openCliTestDB(t)
	// should return without panicking when amount is missing
	setupCardAmountForTag(conn, []string{"SetupCardAmountForTag", "event1"})
}

func TestSetupCardAmountForTag_InvalidAmount(t *testing.T) {
	conn := openCliTestDB(t)
	db.Db_insert_card_with_uid(conn, "k0", "k1", "k2", "k3", "k4", "l1", "p1", "uid1", "event1")
	setupCardAmountForTag(conn, []string{"SetupCardAmountForTag", "event1", "notanumber"})
	// invalid amount -> no receipt added -> balance stays 0
	if bal := getBalance(conn, 1); bal != 0 {
		t.Fatalf("expected balance 0 after invalid amount, got %d", bal)
	}
}

func TestSetupCardAmountForTag_SkipsCardWithExistingReceipts(t *testing.T) {
	conn := openCliTestDB(t)
	db.Db_insert_card_with_uid(conn, "k0", "k1", "k2", "k3", "k4", "l1", "p1", "uid1", "event1")

	// give the card an existing paid receipt
	db.Db_add_card_receipt(conn, 1, "inv", "h1", 999)
	db.Db_set_receipt_paid(conn, "h1", "test")

	// setup should bail (logs error) and not add another receipt
	setupCardAmountForTag(conn, []string{"SetupCardAmountForTag", "event1", "5000"})

	if bal := getBalance(conn, 1); bal != 999 {
		t.Fatalf("expected untouched balance 999, got %d", bal)
	}
}

func TestClearCardBalancesForTag_ZeroesBalances(t *testing.T) {
	conn := openCliTestDB(t)
	db.Db_insert_card_with_uid(conn, "k0", "k1", "k2", "k3", "k4", "l1", "p1", "uid1", "event1")

	// load the card with 3000
	setupCardAmountForTag(conn, []string{"SetupCardAmountForTag", "event1", "3000"})
	if bal := getBalance(conn, 1); bal != 3000 {
		t.Fatalf("setup precondition failed, balance %d", bal)
	}

	clearCardBalancesForTag(conn, []string{"ClearCardBalancesForTag", "event1"})
	if bal := getBalance(conn, 1); bal > 0 {
		t.Fatalf("expected balance <= 0 after clear, got %d", bal)
	}
}

func TestClearCardBalancesForTag_MissingArgs(t *testing.T) {
	conn := openCliTestDB(t)
	clearCardBalancesForTag(conn, []string{"ClearCardBalancesForTag"})
}

func TestProgramBatch_InsertsProgramCard(t *testing.T) {
	conn := openCliTestDB(t)

	programBatch(conn, []string{"ProgramBatch", "batch1", "10", "5000", "24"})

	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM program_cards WHERE group_tag='batch1'").Scan(&count); err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 program_cards row, got %d", count)
	}
}

func TestProgramBatch_WrongArgCount(t *testing.T) {
	conn := openCliTestDB(t)
	programBatch(conn, []string{"ProgramBatch", "batch1", "10"})

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM program_cards").Scan(&count)
	if count != 0 {
		t.Fatalf("expected no rows for wrong arg count, got %d", count)
	}
}

func TestProgramBatch_InvalidNumbers(t *testing.T) {
	conn := openCliTestDB(t)
	programBatch(conn, []string{"ProgramBatch", "batch1", "notnum", "5000", "24"})

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM program_cards").Scan(&count)
	if count != 0 {
		t.Fatalf("expected no rows for invalid max_group_num, got %d", count)
	}
}

func TestWipeCard_WrongArgCount(t *testing.T) {
	conn := openCliTestDB(t)
	// should not panic
	wipeCard(conn, []string{"WipeCard"})
}

func TestWipeCard_InvalidId(t *testing.T) {
	conn := openCliTestDB(t)
	wipeCard(conn, []string{"WipeCard", "notanumber"})
}

func TestWipeCard_ZeroId(t *testing.T) {
	conn := openCliTestDB(t)
	wipeCard(conn, []string{"WipeCard", "0"})
}

func TestWipeCard_ValidCard(t *testing.T) {
	conn := openCliTestDB(t)
	db.Db_insert_card(conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass")

	// wipeCard prints the wipe JSON for a valid, non-wiped card; it must not panic
	wipeCard(conn, []string{"WipeCard", "1"})

	// the CLI wipeCard only displays data; it does not mark the card wiped,
	// so the card should still be retrievable and not wiped
	card, err := db.Db_get_card(conn, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.Wiped != "N" {
		t.Fatalf("expected card still not wiped, got %q", card.Wiped)
	}
}

func TestProcessArgs_UnknownCommand(t *testing.T) {
	conn := openCliTestDB(t)
	// unknown command hits the default branch and just logs a warning
	processArgs(conn, []string{"NopeNotACommand"})
}

func TestProcessArgs_DispatchesProgramBatch(t *testing.T) {
	conn := openCliTestDB(t)
	processArgs(conn, []string{"ProgramBatch", "batchX", "5", "1000", "12"})

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM program_cards WHERE group_tag='batchX'").Scan(&count)
	if count != 1 {
		t.Fatalf("expected ProgramBatch dispatch to insert 1 row, got %d", count)
	}
}
