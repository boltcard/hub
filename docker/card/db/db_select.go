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

	// get card id
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

	// get card id
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
