package db

import (
	"database/sql"
	"os"
	"testing"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	os.Setenv("HOST_DOMAIN", "test.example.com")
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDbInit_SchemaMigratesToLatest(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	version := Db_get_setting(db, "schema_version_number")
	if version != "6" {
		t.Fatalf("expected schema version 6, got %q", version)
	}
}

func TestDbSetAndGetSetting(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	Db_set_setting(db, "test_key", "test_value")
	got := Db_get_setting(db, "test_key")
	if got != "test_value" {
		t.Fatalf("expected %q, got %q", "test_value", got)
	}
}

func TestDbSetSetting_Overwrite(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	Db_set_setting(db, "key1", "value1")
	Db_set_setting(db, "key1", "value2")

	got := Db_get_setting(db, "key1")
	if got != "value2" {
		t.Fatalf("expected %q, got %q", "value2", got)
	}
}

func TestDbGetSetting_Missing(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	got := Db_get_setting(db, "nonexistent")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestDbInsertAndGetCard(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	Db_insert_card(db,
		"0000000000000000", "1111111111111111",
		"2222222222222222", "3333333333333333",
		"4444444444444444", "testlogin", "testpassword")

	card, err := Db_get_card(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if card.Key0_auth != "0000000000000000" {
		t.Fatalf("expected key0 %q, got %q", "0000000000000000", card.Key0_auth)
	}
	if card.Login != "testlogin" {
		t.Fatalf("expected login %q, got %q", "testlogin", card.Login)
	}
}

func TestDbGetCardCount(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	count, err := Db_get_card_count(db)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 cards, got %d", count)
	}

	Db_insert_card(db,
		"0000000000000000", "1111111111111111",
		"2222222222222222", "3333333333333333",
		"4444444444444444", "login1", "pass1")

	count, err = Db_get_card_count(db)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 card, got %d", count)
	}
}

func TestDbTablesExist(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	tables := []string{"settings", "cards", "card_payments", "card_receipts", "program_cards"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Fatalf("table %q not found: %v", table, err)
		}
	}
}

// --- Balance calculation tests ---

func TestDbGetCardBalance_Zero(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	balance := Db_get_card_balance(db, 1)
	if balance != 0 {
		t.Fatalf("expected balance 0, got %d", balance)
	}
}

func TestDbGetCardBalance_ReceiptsOnly(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_add_card_receipt(db, 1, "lnbc1...", "hash1", 500)
	Db_set_receipt_paid(db, "hash1")
	Db_add_card_receipt(db, 1, "lnbc2...", "hash2", 300)
	Db_set_receipt_paid(db, "hash2")

	balance := Db_get_card_balance(db, 1)
	if balance != 800 {
		t.Fatalf("expected balance 800, got %d", balance)
	}
}

func TestDbGetCardBalance_PaymentsAndFeesSubtracted(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_add_card_receipt(db, 1, "lnbc1...", "hash1", 1000)
	Db_set_receipt_paid(db, "hash1")

	payId := Db_add_card_payment(db, 1, 200, "lnbc_pay1")
	Db_update_card_payment_fee(db, payId, 10)

	balance := Db_get_card_balance(db, 1)
	// 1000 - 200 - 10 = 790
	if balance != 790 {
		t.Fatalf("expected balance 790, got %d", balance)
	}
}

func TestDbGetCardBalance_UnpaidReceiptsExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_add_card_receipt(db, 1, "lnbc1...", "hash1", 1000)
	Db_set_receipt_paid(db, "hash1")
	// This receipt is NOT paid — should not count
	Db_add_card_receipt(db, 1, "lnbc2...", "hash2", 500)

	balance := Db_get_card_balance(db, 1)
	if balance != 1000 {
		t.Fatalf("expected balance 1000 (unpaid excluded), got %d", balance)
	}
}

// --- Token operations tests ---

func TestDbSetTokens_ValidLogin(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "mylogin", "mypass")

	err := Db_set_tokens(db, "mylogin", "mypass", "access123", "refresh123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDbSetTokens_InvalidLogin(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "mylogin", "mypass")

	err := Db_set_tokens(db, "wronglogin", "wrongpass", "access123", "refresh123")
	if err == nil {
		t.Fatal("expected error for invalid login, got nil")
	}
}

func TestDbGetCardIdFromAccessToken_Valid(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_set_tokens(db, "login1", "pass1", "tok_access", "tok_refresh")

	cardId := Db_get_card_id_from_access_token(db, "tok_access")
	if cardId == 0 {
		t.Fatal("expected non-zero card_id for valid access token")
	}
}

func TestDbGetCardIdFromAccessToken_Invalid(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	cardId := Db_get_card_id_from_access_token(db, "nonexistent_token")
	if cardId != 0 {
		t.Fatalf("expected 0 for invalid token, got %d", cardId)
	}
}

func TestDbGetCardIdFromAccessToken_WipedExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_set_tokens(db, "login1", "pass1", "tok_access", "tok_refresh")
	Db_wipe_card(db, 1)

	cardId := Db_get_card_id_from_access_token(db, "tok_access")
	if cardId != 0 {
		t.Fatalf("expected 0 for wiped card, got %d", cardId)
	}
}

func TestDbUpdateTokens_Success(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_set_tokens(db, "login1", "pass1", "old_access", "old_refresh")

	ok := Db_update_tokens(db, "old_refresh", "new_refresh", "new_access")
	if !ok {
		t.Fatal("expected token rotation to succeed")
	}

	// Old token should no longer work
	cardId := Db_get_card_id_from_access_token(db, "old_access")
	if cardId != 0 {
		t.Fatal("expected old access token to be invalid")
	}

	// New token should work
	cardId = Db_get_card_id_from_access_token(db, "new_access")
	if cardId == 0 {
		t.Fatal("expected new access token to be valid")
	}
}

func TestDbUpdateTokens_InvalidOldToken(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_set_tokens(db, "login1", "pass1", "access1", "refresh1")

	ok := Db_update_tokens(db, "wrong_refresh", "new_refresh", "new_access")
	if ok {
		t.Fatal("expected token rotation to fail with invalid refresh token")
	}
}

// --- Payment/receipt lifecycle tests ---

func TestDbAddCardReceipt_ReturnsId(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	id := Db_add_card_receipt(db, 1, "lnbc1...", "hash1", 1000)
	if id == 0 {
		t.Fatal("expected non-zero receipt id")
	}
}

func TestDbAddCardPayment_ReturnsId(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	id := Db_add_card_payment(db, 1, 500, "lnbc_pay1")
	if id == 0 {
		t.Fatal("expected non-zero payment id")
	}
}

func TestDbSetReceiptPaid(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_add_card_receipt(db, 1, "lnbc1...", "hash_to_pay", 500)
	// Before paying, balance is 0
	balance := Db_get_card_balance(db, 1)
	if balance != 0 {
		t.Fatalf("expected balance 0 before paying receipt, got %d", balance)
	}

	Db_set_receipt_paid(db, "hash_to_pay")
	balance = Db_get_card_balance(db, 1)
	if balance != 500 {
		t.Fatalf("expected balance 500 after paying receipt, got %d", balance)
	}
}

func TestDbGetPaidPaymentExists(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	// No payments yet
	if Db_get_paid_payment_exists(db, "lnbc_dup") {
		t.Fatal("expected no paid payment to exist")
	}

	Db_add_card_payment(db, 1, 100, "lnbc_dup")
	// Payment exists and is paid by default (paid_flag='Y')
	if !Db_get_paid_payment_exists(db, "lnbc_dup") {
		t.Fatal("expected paid payment to exist")
	}
}

func TestDbGetPaidPaymentExists_UnpaidExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	payId := Db_add_card_payment(db, 1, 100, "lnbc_unpaid")
	Db_update_card_payment_unpaid(db, payId)

	if Db_get_paid_payment_exists(db, "lnbc_unpaid") {
		t.Fatal("expected unpaid payment to not count as existing")
	}
}

func TestDbGetTotalPaidReceipts(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	total := Db_get_total_paid_receipts(db, 1)
	if total != 0 {
		t.Fatalf("expected 0 total paid receipts, got %d", total)
	}

	Db_add_card_receipt(db, 1, "lnbc1...", "hash1", 300)
	Db_set_receipt_paid(db, "hash1")
	Db_add_card_receipt(db, 1, "lnbc2...", "hash2", 700)
	Db_set_receipt_paid(db, "hash2")
	// Unpaid receipt should not count
	Db_add_card_receipt(db, 1, "lnbc3...", "hash3", 999)

	total = Db_get_total_paid_receipts(db, 1)
	if total != 1000 {
		t.Fatalf("expected 1000 total paid receipts, got %d", total)
	}
}

// --- Counter operations tests ---

func TestDbCardCounter_InitialZero(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	counter := Db_get_card_counter(db, 1)
	if counter != 0 {
		t.Fatalf("expected initial counter 0, got %d", counter)
	}
}

func TestDbCardCounter_SetThenGet(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_set_card_counter(db, 1, 42)
	counter := Db_get_card_counter(db, 1)
	if counter != 42 {
		t.Fatalf("expected counter 42, got %d", counter)
	}

	Db_set_card_counter(db, 1, 100)
	counter = Db_get_card_counter(db, 1)
	if counter != 100 {
		t.Fatalf("expected counter 100, got %d", counter)
	}
}

// --- LNURL k1 tests ---

func TestDbLnurlwK1_SetThenGet(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_set_lnurlw_k1(db, 1, "abc123k1", 9999999)

	cardId, expiry := Db_get_lnurlw_k1(db, "abc123k1")
	if cardId != 1 {
		t.Fatalf("expected card_id 1, got %d", cardId)
	}
	if expiry != 9999999 {
		t.Fatalf("expected expiry 9999999, got %d", expiry)
	}
}

func TestDbLnurlwK1_MissingReturnsZero(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	cardId, expiry := Db_get_lnurlw_k1(db, "nonexistent")
	if cardId != 0 {
		t.Fatalf("expected card_id 0 for missing k1, got %d", cardId)
	}
	if expiry != 0 {
		t.Fatalf("expected expiry 0 for missing k1, got %d", expiry)
	}
}

// --- Card wipe tests ---

func TestDbWipeCard_SetsWipedAndReturnsKeys(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "aa00", "bb11", "cc22", "dd33", "ee44", "login1", "pass1")

	keys := Db_wipe_card(db, 1)
	if keys.Key0 != "aa00" || keys.Key1 != "bb11" || keys.Key2 != "cc22" || keys.Key3 != "dd33" || keys.Key4 != "ee44" {
		t.Fatalf("expected original keys, got %+v", keys)
	}

	// Card should be wiped — Db_get_card excludes wiped cards
	_, err := Db_get_card(db, 1)
	if err == nil {
		t.Fatal("expected error getting wiped card")
	}
}

func TestDbWipeCard_CounterExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_set_card_counter(db, 1, 50)
	Db_wipe_card(db, 1)

	// Wiped card should return 0 counter (query has wiped='N' filter)
	counter := Db_get_card_counter(db, 1)
	if counter != 0 {
		t.Fatalf("expected 0 counter for wiped card, got %d", counter)
	}
}

// --- Db_select_card_txs tests ---

func TestDbSelectCardTxs_Empty(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	txs := Db_select_card_txs(db, 1)
	if len(txs) != 0 {
		t.Fatalf("expected 0 txs, got %d", len(txs))
	}
}

func TestDbSelectCardTxs_ReceiptsAndPayments(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	// Add a paid receipt
	Db_add_card_receipt(db, 1, "lnbc1...", "hash1", 500)
	Db_set_receipt_paid(db, "hash1")

	// Add a paid payment
	Db_add_card_payment(db, 1, 200, "lnbc_pay1")

	txs := Db_select_card_txs(db, 1)
	if len(txs) != 2 {
		t.Fatalf("expected 2 txs, got %d", len(txs))
	}

	// Check that receipt has positive amount and payment has negative
	foundPositive := false
	foundNegative := false
	for _, tx := range txs {
		if tx.AmountSats > 0 {
			foundPositive = true
			if tx.AmountSats != 500 {
				t.Fatalf("expected receipt amount 500, got %d", tx.AmountSats)
			}
		}
		if tx.AmountSats < 0 {
			foundNegative = true
			if tx.AmountSats != -200 {
				t.Fatalf("expected payment amount -200, got %d", tx.AmountSats)
			}
		}
	}
	if !foundPositive || !foundNegative {
		t.Fatal("expected both positive (receipt) and negative (payment) txs")
	}
}

func TestDbSelectCardTxs_UnpaidExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	// Unpaid receipt (not calling Db_set_receipt_paid)
	Db_add_card_receipt(db, 1, "lnbc1...", "hash1", 500)

	// Unpaid payment
	payId := Db_add_card_payment(db, 1, 200, "lnbc_pay1")
	Db_update_card_payment_unpaid(db, payId)

	txs := Db_select_card_txs(db, 1)
	if len(txs) != 0 {
		t.Fatalf("expected 0 txs (unpaid excluded), got %d", len(txs))
	}
}

// --- Db_get_card_keys tests ---

func TestDbGetCardKeys_Active(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0a", "k1a", "k2a", "k3a", "k4a", "login1", "pass1")
	Db_insert_card(db, "k0b", "k1b", "k2b", "k3b", "k4b", "login2", "pass2")

	keys := Db_get_card_keys(db)
	if len(keys) != 2 {
		t.Fatalf("expected 2 card lookups, got %d", len(keys))
	}
}

func TestDbGetCardKeys_WipedExcluded(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0a", "k1a", "k2a", "k3a", "k4a", "login1", "pass1")
	Db_wipe_card(db, 1)

	keys := Db_get_card_keys(db)
	if len(keys) != 0 {
		t.Fatalf("expected 0 card lookups (wiped excluded), got %d", len(keys))
	}
}

// --- Db_get_top_cards_by_balance tests ---

func TestDbGetTopCards_Empty(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	top := Db_get_top_cards_by_balance(db, 10)
	if len(top) != 0 {
		t.Fatalf("expected 0 top cards, got %d", len(top))
	}
}

func TestDbGetTopCards_OrderedByBalance(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login2", "pass2")
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login3", "pass3")

	// Fund cards with different amounts
	Db_add_card_receipt(db, 1, "lnbc1", "h1", 100)
	Db_set_receipt_paid(db, "h1")
	Db_add_card_receipt(db, 2, "lnbc2", "h2", 300)
	Db_set_receipt_paid(db, "h2")
	Db_add_card_receipt(db, 3, "lnbc3", "h3", 200)
	Db_set_receipt_paid(db, "h3")

	top := Db_get_top_cards_by_balance(db, 10)
	if len(top) != 3 {
		t.Fatalf("expected 3 top cards, got %d", len(top))
	}
	// Should be DESC order: 300, 200, 100
	if top[0].BalanceSats != 300 {
		t.Fatalf("expected first card balance 300, got %d", top[0].BalanceSats)
	}
	if top[1].BalanceSats != 200 {
		t.Fatalf("expected second card balance 200, got %d", top[1].BalanceSats)
	}
	if top[2].BalanceSats != 100 {
		t.Fatalf("expected third card balance 100, got %d", top[2].BalanceSats)
	}
}

func TestDbGetTopCards_LimitRespected(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login2", "pass2")
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login3", "pass3")

	Db_add_card_receipt(db, 1, "lnbc1", "h1", 100)
	Db_set_receipt_paid(db, "h1")
	Db_add_card_receipt(db, 2, "lnbc2", "h2", 200)
	Db_set_receipt_paid(db, "h2")
	Db_add_card_receipt(db, 3, "lnbc3", "h3", 300)
	Db_set_receipt_paid(db, "h3")

	top := Db_get_top_cards_by_balance(db, 2)
	if len(top) != 2 {
		t.Fatalf("expected 2 top cards with limit=2, got %d", len(top))
	}
}

// --- Db_get_card_id_from_card_uid tests ---

func TestDbGetCardIdFromUid_Valid(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card_with_uid(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1", "AABBCCDD", "tag1")

	cardId := Db_get_card_id_from_card_uid(db, "AABBCCDD")
	if cardId == 0 {
		t.Fatal("expected non-zero card_id")
	}
}

func TestDbGetCardIdFromUid_NotFound(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	cardId := Db_get_card_id_from_card_uid(db, "UNKNOWN")
	if cardId != 0 {
		t.Fatalf("expected 0 for unknown UID, got %d", cardId)
	}
}

// --- Db_update_card_payment_fee tests ---

func TestDbUpdateCardPaymentFee(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_add_card_receipt(db, 1, "lnbc1", "h1", 1000)
	Db_set_receipt_paid(db, "h1")
	payId := Db_add_card_payment(db, 1, 200, "lnbc_pay1")

	// Initially fee is 0, balance = 1000 - 200 - 0 = 800
	balance := Db_get_card_balance(db, 1)
	if balance != 800 {
		t.Fatalf("expected balance 800 before fee update, got %d", balance)
	}

	Db_update_card_payment_fee(db, payId, 10)

	// After fee update, balance = 1000 - 200 - 10 = 790
	balance = Db_get_card_balance(db, 1)
	if balance != 790 {
		t.Fatalf("expected balance 790 after fee update, got %d", balance)
	}
}

// --- Db_update_card_note tests ---

func TestDbUpdateCardNote(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	Db_update_card_note(db, 1, "my test note")

	card, err := Db_get_card(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if card.Note != "my test note" {
		t.Fatalf("expected note 'my test note', got %q", card.Note)
	}
}

func TestDbUpdateCardNote_WipedIgnored(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_update_card_note(db, 1, "original")
	Db_wipe_card(db, 1)

	Db_update_card_note(db, 1, "updated")

	// Card is wiped, so Db_get_card won't find it; query directly
	var note string
	db.QueryRow("SELECT note FROM cards WHERE card_id = 1").Scan(&note)
	if note != "original" {
		t.Fatalf("expected note 'original' (update should be ignored for wiped card), got %q", note)
	}
}

// --- Db_set_card_keys tests ---

func TestDbSetCardKeys(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "old0", "old1", "old2", "old3", "old4", "login1", "pass1")

	Db_set_card_keys(db, 1, "new0", "new1", "new2", "new3", "new4")

	card, err := Db_get_card(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if card.Key0_auth != "new0" {
		t.Fatalf("expected key0 'new0', got %q", card.Key0_auth)
	}
	if card.Key1_enc != "new1" {
		t.Fatalf("expected key1 'new1', got %q", card.Key1_enc)
	}
	if card.Key2_cmac != "new2" {
		t.Fatalf("expected key2 'new2', got %q", card.Key2_cmac)
	}
	if card.Key3 != "new3" {
		t.Fatalf("expected key3 'new3', got %q", card.Key3)
	}
	if card.Key4 != "new4" {
		t.Fatalf("expected key4 'new4', got %q", card.Key4)
	}
}

func TestDbSetCardKeys_WipedIgnored(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card(db, "old0", "old1", "old2", "old3", "old4", "login1", "pass1")
	Db_wipe_card(db, 1)

	Db_set_card_keys(db, 1, "new0", "new1", "new2", "new3", "new4")

	// Query directly since Db_get_card filters wiped cards
	var key0 string
	db.QueryRow("SELECT key0_auth FROM cards WHERE card_id = 1").Scan(&key0)
	if key0 != "old0" {
		t.Fatalf("expected key0 'old0' (set should be ignored for wiped card), got %q", key0)
	}
}

// --- Program Cards tests ---

func TestDbProgramCards_InsertAndSelect(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_program_cards(db, "mysecret", "grouptag1", 10, 5000, 1000, 2000)

	pc := Db_select_program_card_for_secret(db, "mysecret")
	if pc.Secret != "mysecret" {
		t.Fatalf("expected secret 'mysecret', got %q", pc.Secret)
	}
	if pc.GroupTag != "grouptag1" {
		t.Fatalf("expected group_tag 'grouptag1', got %q", pc.GroupTag)
	}
	if pc.MaxGroupNum != 10 {
		t.Fatalf("expected max_group_num 10, got %d", pc.MaxGroupNum)
	}
	if pc.InitialBalance != 5000 {
		t.Fatalf("expected initial_balance 5000, got %d", pc.InitialBalance)
	}
	if pc.CreateTime != 1000 {
		t.Fatalf("expected create_time 1000, got %d", pc.CreateTime)
	}
	if pc.ExpireTime != 2000 {
		t.Fatalf("expected expire_time 2000, got %d", pc.ExpireTime)
	}
}

func TestDbProgramCards_NotFound(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	pc := Db_select_program_card_for_secret(db, "nonexistent")
	if pc.Secret != "" {
		t.Fatalf("expected empty secret for not found, got %q", pc.Secret)
	}
	if pc.GroupTag != "" {
		t.Fatalf("expected empty group_tag for not found, got %q", pc.GroupTag)
	}
}

// --- Db_select_cards_with_group_tag tests ---

func TestDbSelectCardsWithGroupTag_Found(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)
	Db_insert_card_with_uid(db, "k0", "k1", "k2", "k3", "k4", "login1", "pass1", "uid1", "mytag")
	Db_insert_card_with_uid(db, "k0", "k1", "k2", "k3", "k4", "login2", "pass2", "uid2", "mytag")
	Db_insert_card_with_uid(db, "k0", "k1", "k2", "k3", "k4", "login3", "pass3", "uid3", "othertag")

	cards := Db_select_cards_with_group_tag(db, "mytag")
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards with tag 'mytag', got %d", len(cards))
	}
}

func TestDbSelectCardsWithGroupTag_NotFound(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	cards := Db_select_cards_with_group_tag(db, "nonexistent")
	if len(cards) != 0 {
		t.Fatalf("expected 0 cards for unknown tag, got %d", len(cards))
	}
}
