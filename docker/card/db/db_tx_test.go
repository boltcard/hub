package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// openConcurrentTestDB opens a file-backed SQLite database that permits
// multiple concurrent connections. The in-memory ":memory:" DSN used by
// openTestDB gives each pooled connection its own private database, which
// makes it useless for testing the cross-connection locking that
// Db_reserve_card_payment relies on. A temp file with a busy timeout lets
// contending BEGIN IMMEDIATE transactions queue on the SQLite write lock
// instead of failing with SQLITE_BUSY.
func openConcurrentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	os.Setenv("HOST_DOMAIN", "test.example.com")

	dsn := filepath.Join(t.TempDir(), "cards.db") +
		"?_journal=WAL&_timeout=5000&_foreign_keys=1"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// fundCard inserts a card and credits it with a single paid receipt,
// returning the new card's id.
func fundCard(t *testing.T, db *sql.DB, amountSats int) int {
	t.Helper()
	Db_insert_card(db, "k0", "k1", "k2", "k3", "k4", "login", "pass")
	if err := Db_set_tokens(db, "login", "pass", "tok", "refresh"); err != nil {
		t.Fatalf("failed to set tokens: %v", err)
	}
	cardId := Db_get_card_id_from_access_token(db, "tok")
	if cardId == 0 {
		t.Fatal("expected non-zero card_id after insert")
	}
	Db_add_card_receipt(db, cardId, "lnbcreceipt", "fundhash", amountSats)
	Db_set_receipt_paid(db, "fundhash", "test")

	if bal := Db_get_card_balance(db, cardId); bal != amountSats {
		t.Fatalf("expected funded balance %d, got %d", amountSats, bal)
	}
	return cardId
}

// TestReserveCardPayment_Success verifies the happy path: a reservation
// against a sufficiently funded card succeeds, returns a payment id, and
// reduces the card balance by the reserved amount.
func TestReserveCardPayment_Success(t *testing.T) {
	db := openConcurrentTestDB(t)
	Db_init(db)

	cardId := fundCard(t, db, 1000)

	balance, paymentID, err := Db_reserve_card_payment(db, cardId, 1000, 1000, "lnbcpay")
	if err != nil {
		t.Fatalf("expected reservation to succeed, got %v", err)
	}
	if balance != 1000 {
		t.Fatalf("expected reported balance 1000, got %d", balance)
	}
	if paymentID == 0 {
		t.Fatal("expected non-zero payment id")
	}
	if bal := Db_get_card_balance(db, cardId); bal != 0 {
		t.Fatalf("expected balance 0 after reservation, got %d", bal)
	}
}

// TestReserveCardPayment_InsufficientFunds verifies that a reservation
// requiring more than the card balance is rejected, no payment row is
// created, and the balance is unchanged.
func TestReserveCardPayment_InsufficientFunds(t *testing.T) {
	db := openConcurrentTestDB(t)
	Db_init(db)

	cardId := fundCard(t, db, 500)

	balance, paymentID, err := Db_reserve_card_payment(db, cardId, 1000, 1000, "lnbcpay")
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
	if balance != 500 {
		t.Fatalf("expected reported balance 500, got %d", balance)
	}
	if paymentID != 0 {
		t.Fatalf("expected no payment id on failure, got %d", paymentID)
	}
	if bal := Db_get_card_balance(db, cardId); bal != 500 {
		t.Fatalf("expected balance unchanged at 500, got %d", bal)
	}
}

// TestReserveCardPayment_NoDoubleSpend is the core race-condition test for
// the double-spend guard. A card is funded with exactly enough for one
// payment, then many goroutines race to reserve that same amount
// simultaneously. The BEGIN IMMEDIATE transaction in Db_reserve_card_payment
// must serialise these so that exactly one wins and the rest see
// ErrInsufficientFunds — never two successful reservations against a single
// balance.
func TestReserveCardPayment_NoDoubleSpend(t *testing.T) {
	db := openConcurrentTestDB(t)
	Db_init(db)

	const payment = 1000
	cardId := fundCard(t, db, payment)

	const goroutines = 20
	var (
		wg           sync.WaitGroup
		start        = make(chan struct{})
		mu           sync.Mutex
		successes    int
		insufficient int
		otherErrs    []error
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release all goroutines at once to maximise contention
			_, _, err := Db_reserve_card_payment(db, cardId, payment, payment, "lnbcpay")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, ErrInsufficientFunds):
				insufficient++
			default:
				otherErrs = append(otherErrs, err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if len(otherErrs) > 0 {
		t.Fatalf("unexpected errors during concurrent reservations: %v", otherErrs)
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful reservation, got %d", successes)
	}
	if insufficient != goroutines-1 {
		t.Fatalf("expected %d insufficient-funds results, got %d", goroutines-1, insufficient)
	}

	// Exactly one payment row should have been committed and the balance
	// drained to zero — no double spend.
	if bal := Db_get_card_balance(db, cardId); bal != 0 {
		t.Fatalf("expected balance 0 after single reservation, got %d", bal)
	}

	var rows int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM card_payments WHERE card_id = ?", cardId,
	).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("expected exactly 1 card_payments row, got %d", rows)
	}
}
