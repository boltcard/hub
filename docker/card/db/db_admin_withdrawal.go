package db

import (
	"database/sql"
	"time"

	log "github.com/sirupsen/logrus"
)

// AdminWithdrawal is a single admin-initiated payout of node liquidity.
type AdminWithdrawal struct {
	WithdrawalId int
	LnAddress    string
	AmountSats   int
	FeeSats      int
	PaymentHash  string
	Status       string // "pending", "paid" or "failed"
	Timestamp    int
}

type AdminWithdrawals []AdminWithdrawal

// Db_insert_admin_withdrawal records a pending withdrawal and returns its id.
func Db_insert_admin_withdrawal(db_conn *sql.DB, lnAddress string, amountSats int) (int, error) {
	sqlStatement := `INSERT INTO admin_withdrawals (ln_address, amount_sats, status, timestamp)` +
		` VALUES ($1, $2, 'pending', $3);`
	res, err := db_conn.Exec(sqlStatement, lnAddress, amountSats, int(time.Now().Unix()))
	if err != nil {
		log.Error("db_insert_admin_withdrawal error: ", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error("db_insert_admin_withdrawal last insert id error: ", err)
		return 0, err
	}
	return int(id), nil
}

// Db_update_admin_withdrawal_paid marks a withdrawal as paid and records the
// routing fee and payment hash.
func Db_update_admin_withdrawal_paid(db_conn *sql.DB, withdrawalId int, feeSats int, paymentHash string) {
	sqlStatement := `UPDATE admin_withdrawals SET status='paid', fee_sats=$1, payment_hash=$2` +
		` WHERE withdrawal_id=$3;`
	_, err := db_conn.Exec(sqlStatement, feeSats, paymentHash, withdrawalId)
	if err != nil {
		log.Error("db_update_admin_withdrawal_paid error: ", err)
	}
}

// Db_update_admin_withdrawal_failed marks a withdrawal as failed.
func Db_update_admin_withdrawal_failed(db_conn *sql.DB, withdrawalId int) {
	sqlStatement := `UPDATE admin_withdrawals SET status='failed' WHERE withdrawal_id=$1;`
	_, err := db_conn.Exec(sqlStatement, withdrawalId)
	if err != nil {
		log.Error("db_update_admin_withdrawal_failed error: ", err)
	}
}

// Db_select_admin_withdrawals returns recent withdrawals, most recent first.
func Db_select_admin_withdrawals(db_conn *sql.DB, limit int) AdminWithdrawals {
	var withdrawals AdminWithdrawals

	sqlStatement := `SELECT withdrawal_id, ln_address, amount_sats, fee_sats,` +
		` payment_hash, status, timestamp` +
		` FROM admin_withdrawals` +
		` ORDER BY withdrawal_id DESC LIMIT $1;`
	rows, err := db_conn.Query(sqlStatement, limit)
	if err != nil {
		log.Error("db_select_admin_withdrawals query error: ", err)
		return withdrawals
	}
	defer rows.Close()

	for rows.Next() {
		var wd AdminWithdrawal
		err := rows.Scan(
			&wd.WithdrawalId,
			&wd.LnAddress,
			&wd.AmountSats,
			&wd.FeeSats,
			&wd.PaymentHash,
			&wd.Status,
			&wd.Timestamp,
		)
		if err != nil {
			log.Error("db_select_admin_withdrawals scan error: ", err)
			return withdrawals
		}
		withdrawals = append(withdrawals, wd)
	}

	return withdrawals
}
