package db

import (
	"context"
	"database/sql"
	"errors"

	log "github.com/sirupsen/logrus"
)

// ErrInsufficientFunds is returned when a card's balance is too low
// to cover the requested payment amount.
var ErrInsufficientFunds = errors.New("insufficient funds")

// ErrTxLimitExceeded is returned when a payment amount is greater than the
// card's per-transaction limit (tx_limit_sats). A limit of 0 means no limit.
var ErrTxLimitExceeded = errors.New("transaction limit exceeded")

// ErrDayLimitExceeded is returned when a payment would push the card's spend
// over the last 24 hours above its daily limit (day_limit_sats). A limit of 0
// means no limit.
var ErrDayLimitExceeded = errors.New("daily limit exceeded")

// withImmediateTx runs fn inside a BEGIN IMMEDIATE transaction on a
// single pinned connection. BEGIN IMMEDIATE acquires the SQLite write
// lock at transaction start, preventing other writers from interleaving
// between reads and writes within the transaction.
func withImmediateTx(db_conn *sql.DB, fn func(ctx context.Context, conn *sql.Conn) error) error {
	ctx := context.Background()
	conn, err := db_conn.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			conn.ExecContext(ctx, "ROLLBACK")
		}
	}()

	if err := fn(ctx, conn); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	committed = true
	return nil
}

// Db_reserve_card_payment atomically checks the card balance and inserts
// a payment record inside a BEGIN IMMEDIATE transaction. This prevents
// double-spend races where two concurrent requests both read a stale
// balance and both pass the sufficiency check.
//
// requiredBalance is the minimum balance needed (e.g. amount + fee headroom).
// paymentAmount is the amount recorded in the card_payments row.
//
// Returns the actual balance, payment ID, and any error.
// On ErrInsufficientFunds the balance is still returned so the caller
// can choose an appropriate error message.
func Db_reserve_card_payment(db_conn *sql.DB, cardId int, requiredBalance int, paymentAmount int, invoice string) (balance int, paymentID int, err error) {

	err = withImmediateTx(db_conn, func(ctx context.Context, conn *sql.Conn) error {

		// read the card's spending limits under the write lock so the
		// checks below cannot race with a concurrent reservation
		var txLimit, dayLimit int
		limitsSQL := `SELECT tx_limit_sats, day_limit_sats FROM cards WHERE card_id=$1`
		if err := conn.QueryRowContext(ctx, limitsSQL, cardId).Scan(&txLimit, &dayLimit); err != nil {
			return err
		}

		// per-transaction limit (0 = no limit)
		if txLimit > 0 && paymentAmount > txLimit {
			return ErrTxLimitExceeded
		}

		// per-day limit over a rolling 24h window (0 = no limit). Reserved
		// payments count immediately (paid_flag defaults to 'Y') and are
		// reversed to 'N' on failure, so this sum matches spent value.
		if dayLimit > 0 {
			var daySpent int
			daySQL := `SELECT IFNULL(SUM(amount_sats), 0) FROM card_payments
				WHERE paid_flag='Y' AND card_id=$1 AND timestamp >= unixepoch() - 86400`
			if err := conn.QueryRowContext(ctx, daySQL, cardId).Scan(&daySpent); err != nil {
				return err
			}
			if daySpent+paymentAmount > dayLimit {
				return ErrDayLimitExceeded
			}
		}

		// read balance under write lock
		balanceSQL := `SELECT
			IFNULL((SELECT SUM(amount_sats) FROM card_receipts WHERE paid_flag='Y' AND card_id=$1), 0) -
			IFNULL((SELECT SUM(amount_sats) + SUM(fee_sats) FROM card_payments WHERE paid_flag='Y' AND card_id=$1), 0)`
		row := conn.QueryRowContext(ctx, balanceSQL, cardId)
		if err := row.Scan(&balance); err != nil {
			return err
		}

		if balance < requiredBalance {
			return ErrInsufficientFunds
		}

		// reserve funds
		insertSQL := `INSERT INTO card_payments (card_id, amount_sats, ln_invoice,
			timestamp, expire_time)
			VALUES ($1, $2, $3, unixepoch(), unixepoch() + 86400);`
		res, err := conn.ExecContext(ctx, insertSQL, cardId, paymentAmount, invoice)
		if err != nil {
			return err
		}

		count, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			log.Error("db_reserve_card_payment: expected one record to be inserted")
		}

		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		paymentID = int(id)

		return nil
	})

	return balance, paymentID, err
}
