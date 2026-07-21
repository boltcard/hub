package db

import (
	"database/sql"
	"testing"
)

// helper: insert a card and return its id (cards are auto-increment from 1)
func insertTestCard(t *testing.T, db *sql.DB, login string) {
	t.Helper()
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", login, "pass")
}

func TestDbSelectCardPayments_OrderedAndEmpty(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	// no card -> empty
	if got := Db_select_card_payments(db, 1); len(got) != 0 {
		t.Fatalf("expected no payments, got %d", len(got))
	}

	insertTestCard(t, db, "login1")
	p1 := Db_add_card_payment(db, 1, 100, "inv1")
	Db_update_card_payment_fee(db, p1, 5)
	Db_add_card_payment(db, 1, 200, "inv2")

	payments := Db_select_card_payments(db, 1)
	if len(payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(payments))
	}
	// ordered by card_payment_id DESC, so newest (inv2) first
	if payments[0].AmountSats != 200 {
		t.Fatalf("expected newest payment first (200), got %d", payments[0].AmountSats)
	}
	if payments[1].AmountSats != 100 || payments[1].FeeSats != 5 {
		t.Fatalf("expected second payment 100/fee 5, got %d/%d", payments[1].AmountSats, payments[1].FeeSats)
	}
}

func TestDbSelectCardReceipts_LimitAndAll(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	insertTestCard(t, db, "login1")

	Db_add_card_receipt(db, 1, "lnbc1", "h1", 1000)
	Db_add_card_receipt(db, 1, "lnbc2", "h2", 2000)
	Db_add_card_receipt(db, 1, "lnbc3", "h3", 3000)

	all := Db_select_card_receipts(db, 1, 0)
	if len(all) != 3 {
		t.Fatalf("expected 3 receipts with limit 0, got %d", len(all))
	}
	// ordered DESC by id -> newest (h3) first
	if all[0].PaymentHash != "h3" {
		t.Fatalf("expected newest receipt first (h3), got %q", all[0].PaymentHash)
	}

	limited := Db_select_card_receipts(db, 1, 2)
	if len(limited) != 2 {
		t.Fatalf("expected 2 receipts with limit 2, got %d", len(limited))
	}
	if limited[0].PaymentHash != "h3" || limited[1].PaymentHash != "h2" {
		t.Fatalf("unexpected limited ordering: %q, %q", limited[0].PaymentHash, limited[1].PaymentHash)
	}
}

func TestDbSelectAllCards_ExcludesWipedAndComputesBalance(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	// empty
	if got := Db_select_all_cards(db); len(got) != 0 {
		t.Fatalf("expected no cards, got %d", len(got))
	}

	insertTestCard(t, db, "login1")
	insertTestCard(t, db, "login2")

	// card 1 has a paid receipt of 1000 and a paid payment of 200
	Db_add_card_receipt(db, 1, "lnbc1", "h1", 1000)
	Db_set_receipt_paid(db, "h1", "test")
	Db_add_card_payment(db, 1, 200, "inv1")

	// wipe card 2
	Db_wipe_card(db, 2)

	cards := Db_select_all_cards(db)
	if len(cards) != 1 {
		t.Fatalf("expected 1 non-wiped card, got %d", len(cards))
	}
	if cards[0].CardId != 1 {
		t.Fatalf("expected card id 1, got %d", cards[0].CardId)
	}
	if cards[0].BalanceSats != 800 {
		t.Fatalf("expected balance 800 (1000-200), got %d", cards[0].BalanceSats)
	}
}

func TestDbSelectAllSettings_Ordered(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	Db_set_setting(db, "zzz_key", "z")
	Db_set_setting(db, "aaa_key", "a")

	settings := Db_select_all_settings(db)
	if len(settings) == 0 {
		t.Fatal("expected settings, got none")
	}

	// verify our two keys are present and the list is sorted by name
	var prev string
	foundA, foundZ := false, false
	for _, s := range settings {
		if prev != "" && s.Name < prev {
			t.Fatalf("settings not sorted: %q came after %q", s.Name, prev)
		}
		prev = s.Name
		if s.Name == "aaa_key" {
			foundA = true
			if s.Value != "a" {
				t.Fatalf("aaa_key value = %q, want a", s.Value)
			}
		}
		if s.Name == "zzz_key" {
			foundZ = true
		}
	}
	if !foundA || !foundZ {
		t.Fatalf("expected both inserted settings present (a=%v z=%v)", foundA, foundZ)
	}
}

func TestDbGetTotalCardBalance(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	if got := Db_get_total_card_balance(db); got != 0 {
		t.Fatalf("expected 0 total balance, got %d", got)
	}

	insertTestCard(t, db, "login1")
	insertTestCard(t, db, "login2")

	// both cards lnurlw_enable defaults to 'Y'
	Db_add_card_receipt(db, 1, "lnbc1", "h1", 1000)
	Db_set_receipt_paid(db, "h1", "test")
	Db_add_card_receipt(db, 2, "lnbc2", "h2", 500)
	Db_set_receipt_paid(db, "h2", "test")
	Db_add_card_payment(db, 1, 200, "inv1")

	// total = (1000 - 200) + 500 = 1300
	if got := Db_get_total_card_balance(db); got != 1300 {
		t.Fatalf("expected total balance 1300, got %d", got)
	}
}

func TestDbGetTableCounts(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	insertTestCard(t, db, "login1")
	Db_add_card_receipt(db, 1, "lnbc1", "h1", 1000)

	counts, err := Db_get_table_counts(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byName := make(map[string]int)
	for _, c := range counts {
		byName[c.Name] = c.Count
	}
	if byName["cards"] != 1 {
		t.Fatalf("expected 1 card, got %d", byName["cards"])
	}
	if byName["card_receipts"] != 1 {
		t.Fatalf("expected 1 receipt, got %d", byName["card_receipts"])
	}
	if _, ok := byName["settings"]; !ok {
		t.Fatal("expected settings table in counts")
	}
}

func TestDbGetCardNoteByInvoice(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	insertTestCard(t, db, "login1")
	Db_update_card_note(db, 1, "Alice card")

	// unpaid payment -> no note returned
	Db_add_card_payment(db, 1, 100, "inv_unpaid")
	Db_update_card_payment_unpaid(db, 1)
	if got := Db_get_card_note_by_invoice(db, "inv_unpaid"); got != "" {
		t.Fatalf("expected empty note for unpaid payment, got %q", got)
	}

	// paid payment (default paid_flag='Y') -> note returned
	Db_add_card_payment(db, 1, 100, "inv_paid")
	if got := Db_get_card_note_by_invoice(db, "inv_paid"); got != "Alice card" {
		t.Fatalf("expected note 'Alice card', got %q", got)
	}

	// unknown invoice -> empty
	if got := Db_get_card_note_by_invoice(db, "does_not_exist"); got != "" {
		t.Fatalf("expected empty note for unknown invoice, got %q", got)
	}
}

func TestDbGetCardLnurlwEnable(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	insertTestCard(t, db, "login1")

	// default enabled
	if got := Db_get_card_lnurlw_enable(db, 1); got != "Y" {
		t.Fatalf("expected default lnurlw_enable 'Y', got %q", got)
	}

	Db_update_card_without_pin(db, 1, 1000, 5000, "N", 0, "N")
	if got := Db_get_card_lnurlw_enable(db, 1); got != "N" {
		t.Fatalf("expected lnurlw_enable 'N' after update, got %q", got)
	}

	// missing card -> empty
	if got := Db_get_card_lnurlw_enable(db, 999); got != "" {
		t.Fatalf("expected empty for missing card, got %q", got)
	}
}

// TestDbInsertCard_EnablesWithdrawalsOnLegacyDefault reproduces the upgraded-hub
// bug: the cards-table schema default for lnurlw_enable was changed from 'N' to
// 'Y' (commit 2cbd45e), but CREATE TABLE IF NOT EXISTS never re-applies a default
// on hubs that already had the table. Their column default stays 'N', so newly
// programmed cards were inserted disabled and would not withdraw. The insert must
// set lnurlw_enable='Y' explicitly, independent of the column default.
func TestDbInsertCard_EnablesWithdrawalsOnLegacyDefault(t *testing.T) {
	db := openTestDB(t)

	// Recreate the deployed-hub condition: cards table with the legacy
	// lnurlw_enable DEFAULT 'N'. Only the columns the insert/getter touch.
	_, err := db.Exec(`CREATE TABLE cards (
		card_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		key0_auth CHAR(32) NOT NULL,
		key1_enc CHAR(32) NOT NULL,
		key2_cmac CHAR(32) NOT NULL,
		key3 CHAR(32) NOT NULL,
		key4 CHAR(32) NOT NULL,
		login CHAR(32) NOT NULL,
		password CHAR(32) NOT NULL,
		ln_address CHAR(32) NOT NULL DEFAULT '',
		lnurlw_enable CHAR(1) NOT NULL DEFAULT 'N',
		wiped CHAR(1) NOT NULL DEFAULT 'N'
	);`)
	if err != nil {
		t.Fatal(err)
	}

	Db_insert_card(db,
		"0000000000000000", "1111111111111111",
		"2222222222222222", "3333333333333333",
		"4444444444444444", "testlogin", "testpassword")

	if got := Db_get_card_lnurlw_enable(db, 1); got != "Y" {
		t.Fatalf("newly programmed card should be enabled; got lnurlw_enable=%q", got)
	}
}

// TestDbInsertCardWithUid_EnablesWithdrawalsOnLegacyDefault is the batch-programming
// counterpart to the test above; the batch (/batch) endpoint uses this insert path.
func TestDbInsertCardWithUid_EnablesWithdrawalsOnLegacyDefault(t *testing.T) {
	db := openTestDB(t)

	_, err := db.Exec(`CREATE TABLE cards (
		card_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		key0_auth CHAR(32) NOT NULL,
		key1_enc CHAR(32) NOT NULL,
		key2_cmac CHAR(32) NOT NULL,
		key3 CHAR(32) NOT NULL,
		key4 CHAR(32) NOT NULL,
		login CHAR(32) NOT NULL,
		password CHAR(32) NOT NULL,
		uid VARCHAR(14) NOT NULL DEFAULT '',
		group_tag TEXT NOT NULL DEFAULT '',
		ln_address CHAR(32) NOT NULL DEFAULT '',
		lnurlw_enable CHAR(1) NOT NULL DEFAULT 'N',
		wiped CHAR(1) NOT NULL DEFAULT 'N'
	);`)
	if err != nil {
		t.Fatal(err)
	}

	Db_insert_card_with_uid(db,
		"0000000000000000", "1111111111111111",
		"2222222222222222", "3333333333333333",
		"4444444444444444", "testlogin", "testpassword",
		"04AABBCCDDEE80", "grp1")

	if got := Db_get_card_lnurlw_enable(db, 1); got != "Y" {
		t.Fatalf("newly programmed card should be enabled; got lnurlw_enable=%q", got)
	}
}

func TestDbUpdateCardWithPin(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	insertTestCard(t, db, "login1")

	Db_update_card_with_pin(db, 1, 1234, 5678, "Y", "4321", 999, "Y")

	// verify a couple of the written columns via existing getters / raw query
	var txLimit, dayLimit, pinLimit int
	var pinEnable, pinNumber, lnurlw string
	row := db.QueryRow(`SELECT tx_limit_sats, day_limit_sats, pin_enable, pin_number, pin_limit_sats, lnurlw_enable FROM cards WHERE card_id=1`)
	if err := row.Scan(&txLimit, &dayLimit, &pinEnable, &pinNumber, &pinLimit, &lnurlw); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if txLimit != 1234 || dayLimit != 5678 || pinEnable != "Y" || pinNumber != "4321" || pinLimit != 999 || lnurlw != "Y" {
		t.Fatalf("unexpected row after Db_update_card_with_pin: %d %d %q %q %d %q",
			txLimit, dayLimit, pinEnable, pinNumber, pinLimit, lnurlw)
	}
}

func TestDbUpdateCardWithoutPin_LeavesPinNumber(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	insertTestCard(t, db, "login1")

	// set a pin first
	Db_update_card_with_pin(db, 1, 10, 20, "Y", "1111", 50, "Y")
	// update without pin should not touch pin_number
	Db_update_card_without_pin(db, 1, 30, 40, "N", 60, "N")

	var pinNumber, pinEnable string
	var txLimit int
	row := db.QueryRow(`SELECT pin_number, pin_enable, tx_limit_sats FROM cards WHERE card_id=1`)
	if err := row.Scan(&pinNumber, &pinEnable, &txLimit); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if pinNumber != "1111" {
		t.Fatalf("expected pin_number preserved as 1111, got %q", pinNumber)
	}
	if pinEnable != "N" || txLimit != 30 {
		t.Fatalf("expected pin_enable N / tx_limit 30, got %q / %d", pinEnable, txLimit)
	}
}

func TestDbUpdateReceiptPaid(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	insertTestCard(t, db, "login1")

	rid := Db_add_card_receipt(db, 1, "lnbc1", "h1", 1000)
	// before: not counted toward balance
	if bal := Db_get_card_balance(db, 1); bal != 0 {
		t.Fatalf("expected balance 0 before receipt paid, got %d", bal)
	}

	Db_update_receipt_paid(db, rid)
	if bal := Db_get_card_balance(db, 1); bal != 1000 {
		t.Fatalf("expected balance 1000 after receipt paid, got %d", bal)
	}
}

func TestAdminWithdrawalLifecycle(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	id, err := Db_insert_admin_withdrawal(db, "alice@example.com", 25000)
	if err != nil {
		t.Fatalf("insert error: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	rows := Db_select_admin_withdrawals(db, 10)
	if len(rows) != 1 {
		t.Fatalf("expected 1 withdrawal, got %d", len(rows))
	}
	if rows[0].Status != "pending" || rows[0].AmountSats != 25000 {
		t.Fatalf("unexpected pending row: %+v", rows[0])
	}

	Db_update_admin_withdrawal_paid(db, id, 7, "paymenthash")
	rows = Db_select_admin_withdrawals(db, 10)
	if rows[0].Status != "paid" || rows[0].FeeSats != 7 || rows[0].PaymentHash != "paymenthash" {
		t.Fatalf("unexpected paid row: %+v", rows[0])
	}

	id2, _ := Db_insert_admin_withdrawal(db, "bob@example.com", 100)
	Db_update_admin_withdrawal_failed(db, id2)
	rows = Db_select_admin_withdrawals(db, 10)
	// Most recent first
	if rows[0].Status != "failed" {
		t.Fatalf("expected most recent to be failed, got %+v", rows[0])
	}
}
