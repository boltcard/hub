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
