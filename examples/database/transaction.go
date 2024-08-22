package database

import (
	"strconv"
	"strings"
)

type Tx struct {
	ID                     int64
	TxID                   string
	Status                 int // 0 pending 1 confirmed 2 failed
	UnsignedRawTransaction string
	SenderAccountIDs       string
	SignedRawTransaction   string
}

func InsertTx(
	txID string,
	status int,
	unsignedRawTransaction string,
	senderAccountIDs []int64,
	signedRawTransaction string) (int64, error) {

	stmt, err := db.Prepare("INSERT INTO tx (tx_id,status,unsigned_transaction,sender_account_ids,signed_transaction) VALUES (?,?,?,?,?)")
	if err != nil {
		return -1, err
	}

	senderAccountIDStrs := make([]string, len(senderAccountIDs))
	for i := 0; i < len(senderAccountIDs); i++ {
		senderAccountIDStrs[i] = strconv.Itoa(int(senderAccountIDs[i]))
	}

	result, err := stmt.Exec(txID, status, unsignedRawTransaction, strings.Join(senderAccountIDStrs, ","), signedRawTransaction)
	if err != nil {
		return -1, err
	}
	return result.LastInsertId()
}
func LoadPendingTransactions() ([]*Tx, error) {
	rows, err := db.Query("SELECT ID,tx_id,status,unsigned_transaction,sender_account_ids,signed_transaction FROM tx WHERE status = 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	txs := []*Tx{}
	for rows.Next() {
		var id int64
		var txID string
		var status int
		var unsignedRawTransaction string
		var senderAccountIDsStr string
		var signedRawTransaction string
		err = rows.Scan(&id, &txID, &status, &unsignedRawTransaction, &senderAccountIDsStr, &signedRawTransaction)
		if err != nil {
			return nil, err
		}
		txs = append(txs, &Tx{
			ID:                     id,
			TxID:                   txID,
			Status:                 status,
			UnsignedRawTransaction: unsignedRawTransaction,
			SenderAccountIDs:       senderAccountIDsStr,
			SignedRawTransaction:   signedRawTransaction,
		})
	}
	return txs, nil
}
func MarkTxConfirmed(txID string) error {
	stmt, err := db.Prepare("UPDATE tx SET status = ? WHERE tx_id = ?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(1, txID)
	if err != nil {
		return err
	}
	return nil
}
