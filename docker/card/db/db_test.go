package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
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

func TestDbInit_SchemaMigratesToVersion5(t *testing.T) {
	db := openTestDB(t)
	Db_init(db)

	version := Db_get_setting(db, "schema_version_number")
	if version != "5" {
		t.Fatalf("expected schema version 5, got %q", version)
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
