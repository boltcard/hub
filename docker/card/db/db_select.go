package db

import (
	"card/util"
	"database/sql"
)

type CardReceipt struct {
	CardReceiptId  int
	PaymentRequest string
	PaymentHash    string
	AmountSats     int
	IsPaid         string
	Timestamp      int
	ExpireTime     int
}

type CardReceipts []CardReceipt

// Db_select_card_receipts returns card receipts ordered by most recent first.
// Pass limit=0 to return all receipts.
func Db_select_card_receipts(db_conn *sql.DB, card_id int, limit int) (result CardReceipts) {
	var cardReceipts CardReceipts

	var rows *sql.Rows
	var err error

	if limit > 0 {
		sqlStatement := `SELECT card_receipt_id, ln_invoice,` +
			` r_hash_hex, amount_sats, paid_flag,` +
			` timestamp, expire_time` +
			` FROM card_receipts` +
			` WHERE card_receipts.card_id = $1` +
			` ORDER BY card_receipt_id DESC LIMIT $2;`
		rows, err = db_conn.Query(sqlStatement, card_id, limit)
	} else {
		sqlStatement := `SELECT card_receipt_id, ln_invoice,` +
			` r_hash_hex, amount_sats, paid_flag,` +
			` timestamp, expire_time` +
			` FROM card_receipts` +
			` WHERE card_receipts.card_id = $1` +
			` ORDER BY card_receipt_id DESC;`
		rows, err = db_conn.Query(sqlStatement, card_id)
	}
	util.CheckAndPanic(err)
	defer rows.Close()

	for rows.Next() {
		var cardReceipt CardReceipt

		err := rows.Scan(
			&cardReceipt.CardReceiptId,
			&cardReceipt.PaymentRequest,
			&cardReceipt.PaymentHash,
			&cardReceipt.AmountSats,
			&cardReceipt.IsPaid,
			&cardReceipt.Timestamp,
			&cardReceipt.ExpireTime)
		util.CheckAndPanic(err)

		cardReceipts = append(cardReceipts, cardReceipt)
	}

	return cardReceipts
}

type CardPayment struct {
	CardPaymentId int
	AmountSats    int
	FeeSats       int
	IsPaid        string
	Timestamp     int
	ExpireTime    int
}

type CardPayments []CardPayment

func Db_select_card_payments(db_conn *sql.DB, card_id int) (result CardPayments) {
	var cardPayments CardPayments

	sqlStatement := `SELECT card_payment_id,` +
		` amount_sats, fee_sats, paid_flag,` +
		` timestamp, expire_time` +
		` FROM card_payments` +
		` WHERE card_payments.card_id = $1` +
		` ORDER BY card_payment_id DESC;`
	rows, err := db_conn.Query(sqlStatement, card_id)
	util.CheckAndPanic(err)
	defer rows.Close()

	for rows.Next() {
		var cardPayment CardPayment

		err := rows.Scan(
			&cardPayment.CardPaymentId,
			&cardPayment.AmountSats,
			&cardPayment.FeeSats,
			&cardPayment.IsPaid,
			&cardPayment.Timestamp,
			&cardPayment.ExpireTime)
		util.CheckAndPanic(err)

		cardPayments = append(cardPayments, cardPayment)
	}

	return cardPayments
}

type CardTx struct {
	ReceiptId  int
	PaymentId  int
	Timestamp  int
	AmountSats int
	FeeSats    int
}

type CardTxs []CardTx

func Db_select_card_txs(db_conn *sql.DB, card_id int) (result CardTxs) {
	var cardTxs CardTxs

	// get card txs
	sqlStatement := `SELECT card_receipt_id, 0, timestamp, amount_sats, fee_sats` +
		` FROM card_receipts` +
		` WHERE card_receipts.card_id = $1 AND card_receipts.paid_flag='Y'` +
		` UNION` +
		` SELECT 0, card_payment_id, timestamp, -amount_sats, -fee_sats` +
		` FROM card_payments` +
		` WHERE card_payments.card_id = $1 AND card_payments.paid_flag='Y'` +
		` ORDER BY timestamp;`
	rows, err := db_conn.Query(sqlStatement, card_id)
	util.CheckAndPanic(err)
	defer rows.Close()

	for rows.Next() {
		var cardTx CardTx

		err := rows.Scan(
			&cardTx.ReceiptId,
			&cardTx.PaymentId,
			&cardTx.Timestamp,
			&cardTx.AmountSats,
			&cardTx.FeeSats)
		util.CheckAndPanic(err)

		cardTxs = append(cardTxs, cardTx)
	}

	return cardTxs
}

type CardIdOnly struct {
	CardId int
}

type Cards []CardIdOnly

func Db_select_cards_with_group_tag(db_conn *sql.DB, group_tag string) (result Cards) {
	var cards Cards

	// get card id
	sqlStatement := `SELECT card_id` +
		` FROM cards` +
		` WHERE group_tag = $1;`
	rows, err := db_conn.Query(sqlStatement, group_tag)
	util.CheckAndPanic(err)
	defer rows.Close()

	for rows.Next() {
		var cardIdOnly CardIdOnly

		err := rows.Scan(
			&cardIdOnly.CardId)
		util.CheckAndPanic(err)

		cards = append(cards, cardIdOnly)
	}

	return cards
}

// program_card_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
// secret TEXT NOT NULL DEFAULT '',
// group_tag TEXT NOT NULL DEFAULT '',
// max_group_num INTEGER NOT NULL DEFAULT 0,
// initial_balance INTEGER NOT NULL DEFAULT 0,
// create_time INTEGER NOT NULL,
// expire_time INTEGER NOT NULL

type ProgramCard struct {
	ProgramCardId  int
	Secret         string
	GroupTag       string
	MaxGroupNum    int
	InitialBalance int
	CreateTime     int
	ExpireTime     int
}

func Db_select_program_card_for_secret(db_conn *sql.DB, secret string) (result ProgramCard) {
	var programCard ProgramCard

	// get card id
	sqlStatement := `SELECT secret, group_tag, max_group_num, initial_balance, create_time, expire_time` +
		` FROM program_cards WHERE secret = $1;`
	rows, err := db_conn.Query(sqlStatement, secret)
	util.CheckAndPanic(err)
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(
			&programCard.Secret,
			&programCard.GroupTag,
			&programCard.MaxGroupNum,
			&programCard.InitialBalance,
			&programCard.CreateTime,
			&programCard.ExpireTime)
		util.CheckAndPanic(err)
	}

	return programCard
}
