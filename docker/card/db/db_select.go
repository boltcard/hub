package db

import (
	"card/util"

	_ "github.com/mattn/go-sqlite3"
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

func Db_select_card_receipts_with_limit(card_id int, limit int) (result CardReceipts) {
	var cardReceipts CardReceipts

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// get card id
	sqlStatement := `SELECT card_receipt_id, ln_invoice,` +
		` r_hash_hex, amount_sats, paid_flag,` +
		` timestamp, expire_time` +
		` FROM card_receipts` +
		` WHERE card_receipts.card_id = $1` +
		` ORDER BY card_receipt_id DESC LIMIT $2;`
	rows, err := db.Query(sqlStatement, card_id, limit)
	util.Check(err)

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
		util.Check(err)

		cardReceipts = append(cardReceipts, cardReceipt)
	}

	return cardReceipts
}

func Db_select_card_receipts(card_id int) (result CardReceipts) {
	var cardReceipts CardReceipts

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	sqlStatement := `SELECT card_receipt_id, ln_invoice,` +
		` r_hash_hex, amount_sats, paid_flag,` +
		` timestamp, expire_time` +
		` FROM card_receipts` +
		` WHERE card_receipts.card_id = $1` +
		` ORDER BY card_receipt_id DESC;`
	rows, err := db.Query(sqlStatement, card_id)
	util.Check(err)

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
		util.Check(err)

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

func Db_select_card_payments(card_id int) (result CardPayments) {
	var cardPayments CardPayments

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	sqlStatement := `SELECT card_payment_id,` +
		` amount_sats, paid_flag,` +
		` timestamp, expire_time` +
		` FROM card_payments` +
		` WHERE card_payments.card_id = $1` +
		` ORDER BY card_payment_id DESC;`
	rows, err := db.Query(sqlStatement, card_id)
	util.Check(err)

	for rows.Next() {
		var cardPayment CardPayment

		err := rows.Scan(
			&cardPayment.CardPaymentId,
			&cardPayment.AmountSats,
			&cardPayment.IsPaid,
			&cardPayment.Timestamp,
			&cardPayment.ExpireTime)
		util.Check(err)

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

func Db_select_card_txs(card_id int) (result CardTxs) {
	var cardTxs CardTxs

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// get card txs
	sqlStatement := `SELECT card_receipt_id, 0, timestamp, amount_sats, fee_sats` +
		` FROM card_receipts` +
		` WHERE card_receipts.card_id = $1 AND card_receipts.paid_flag='Y'` +
		` UNION` +
		` SELECT 0, card_payment_id, timestamp, -amount_sats, -fee_sats` +
		` FROM card_payments` +
		` WHERE card_payments.card_id = $1 AND card_payments.paid_flag='Y'` +
		` ORDER BY timestamp;`
	rows, err := db.Query(sqlStatement, card_id)
	util.Check(err)

	for rows.Next() {
		var cardTx CardTx

		err := rows.Scan(
			&cardTx.ReceiptId,
			&cardTx.PaymentId,
			&cardTx.Timestamp,
			&cardTx.AmountSats,
			&cardTx.FeeSats)
		util.Check(err)

		cardTxs = append(cardTxs, cardTx)
	}

	return cardTxs
}

type CardIdOnly struct {
	CardId int
}

type Cards []CardIdOnly

func Db_select_cards_with_group_tag(group_tag string) (result Cards) {
	var cards Cards

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// get card id
	sqlStatement := `SELECT card_id` +
		` FROM cards` +
		` WHERE group_tag = $1;`
	rows, err := db.Query(sqlStatement, group_tag)
	util.Check(err)

	for rows.Next() {
		var cardIdOnly CardIdOnly

		err := rows.Scan(
			&cardIdOnly.CardId)
		util.Check(err)

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

func Db_select_program_card_for_secret(secret string) (result ProgramCard) {
	var programCard ProgramCard

	// open a database connection
	db, err := Open()
	util.Check(err)
	defer db.Close()

	// get card id
	sqlStatement := `SELECT secret, group_tag, max_group_num, initial_balance, create_time, expire_time` +
		` FROM program_cards WHERE secret = $1;`
	rows, err := db.Query(sqlStatement, secret)
	util.Check(err)

	if rows.Next() {
		err := rows.Scan(
			&programCard.Secret,
			&programCard.GroupTag,
			&programCard.MaxGroupNum,
			&programCard.InitialBalance,
			&programCard.CreateTime,
			&programCard.ExpireTime)
		util.Check(err)
	}

	return programCard
}
