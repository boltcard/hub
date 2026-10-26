package db

import (
	"errors"
	"testing"
)

// Columns such as cards.access_token, cards.refresh_token and
// program_cards.secret default to '', so an empty supplied value must never
// be used to look a row up.

func TestDbGetCardIdFromAccessToken_EmptyNeverMatches(t *testing.T) {
	db_conn := openTestDB(t)
	Db_init(db_conn)
	// a card that never called /auth keeps access_token = ''
	Db_insert_card(db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")

	if id := Db_get_card_id_from_access_token(db_conn, ""); id != 0 {
		t.Fatalf("empty access token matched card %d", id)
	}
}

func TestDbUpdateTokens_EmptyRefreshTokenNeverMatches(t *testing.T) {
	db_conn := openTestDB(t)
	Db_init(db_conn)
	Db_insert_card(db_conn, "k0", "k1", "k2", "k3", "k4", "login1", "pass1")
	Db_insert_card(db_conn, "k0", "k1", "k2", "k3", "k4", "login2", "pass2")

	if Db_update_tokens(db_conn, "", "newrefresh", "newaccess") {
		t.Fatal("empty refresh token was accepted")
	}

	// no card may have been given the new tokens
	var n int
	if err := db_conn.QueryRow(
		`SELECT COUNT(*) FROM cards WHERE access_token = 'newaccess'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("empty refresh token overwrote tokens on %d cards", n)
	}
}

func TestDbSetTokens_EmptyCredentialsNeverMatch(t *testing.T) {
	db_conn := openTestDB(t)
	Db_init(db_conn)
	Db_insert_card(db_conn, "k0", "k1", "k2", "k3", "k4", "", "")

	if err := Db_set_tokens(db_conn, "", "", "a", "r"); err == nil {
		t.Fatal("empty login/password was accepted")
	}
}

func TestDbDispenseBatchCard_EmptySecretNeverMatches(t *testing.T) {
	db_conn := openTestDB(t)
	Db_init(db_conn)
	// only an imported or hand-edited database would hold an empty secret
	insertTestBatch(t, db_conn, "", 10, 5000)

	_, _, err := Db_dispense_batch_card(db_conn, "", "", testBatchKeys(), "login", "pass")
	if !errors.Is(err, ErrBatchNotFound) {
		t.Fatalf("expected ErrBatchNotFound for empty secret, got %v", err)
	}
	if n := countCards(t, db_conn); n != 0 {
		t.Fatalf("expected no card dispensed, got %d", n)
	}
}
