package db

import (
	"database/sql"
	"testing"
	"time"
)

// insertUnfundedCard inserts a card with known keys and returns its id.
func insertUnfundedCard(t *testing.T, db *sql.DB) int {
	t.Helper()
	Db_insert_card(db, "wk0", "wk1enc", "wk2cmac", "wk3", "wk4", "wlogin", "wpass")
	if err := Db_set_tokens(db, "wlogin", "wpass", "wtok", "wref"); err != nil {
		t.Fatalf("set tokens: %v", err)
	}
	id := Db_get_card_id_from_access_token(db, "wtok")
	if id == 0 {
		t.Fatal("expected non-zero card id")
	}
	return id
}

// TestCardWipeSecret_ValidReturnsKeys verifies a wipe secret resolves the
// card's keys even after the card has been wiped (wiped='Y'), which the normal
// getters filter out.
func TestCardWipeSecret_ValidReturnsKeys(t *testing.T) {
	db := openConcurrentTestDB(t)
	Db_init(db)
	id := insertUnfundedCard(t, db)

	Db_wipe_card(db, id) // marks wiped='Y'
	Db_set_card_wipe_secret(db, id, "secretAAA", time.Now().Unix()+3600)

	keys := Db_get_card_keys_for_wipe_secret(db, "secretAAA")
	if keys.Key0 != "wk0" || keys.Key1 != "wk1enc" || keys.Key2 != "wk2cmac" ||
		keys.Key3 != "wk3" || keys.Key4 != "wk4" {
		t.Fatalf("unexpected wipe keys: %+v", keys)
	}
}

// TestCardWipeSecret_ExpiredReturnsEmpty verifies an expired secret resolves
// no keys.
func TestCardWipeSecret_ExpiredReturnsEmpty(t *testing.T) {
	db := openConcurrentTestDB(t)
	Db_init(db)
	id := insertUnfundedCard(t, db)
	Db_set_card_wipe_secret(db, id, "secretEXP", time.Now().Unix()-10)

	if keys := Db_get_card_keys_for_wipe_secret(db, "secretEXP"); keys.Key0 != "" {
		t.Fatalf("expected no keys for expired secret, got %+v", keys)
	}
}

// TestCardWipeSecret_UnknownReturnsEmpty verifies an unknown secret resolves
// no keys (and an empty secret never matches).
func TestCardWipeSecret_UnknownReturnsEmpty(t *testing.T) {
	db := openConcurrentTestDB(t)
	Db_init(db)
	insertUnfundedCard(t, db)

	if keys := Db_get_card_keys_for_wipe_secret(db, "does-not-exist"); keys.Key0 != "" {
		t.Fatalf("expected no keys for unknown secret, got %+v", keys)
	}
	if keys := Db_get_card_keys_for_wipe_secret(db, ""); keys.Key0 != "" {
		t.Fatalf("expected no keys for empty secret, got %+v", keys)
	}
}
