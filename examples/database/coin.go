package database

import (
	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
)

type Coin struct {
	ID        int64
	AccountID int64
	Status    int // 0-immature 1-spendable 2-spent 3-confirmed 4-invalid
	*abelian.Coin
}

func InsertCoin(
	accountID int64,
	txVersion uint32,
	txID string,
	index uint8,
	blockHash string,
	blockHeight int64,
	value int64,
	isCoinbase bool,
	data []byte) (int64, error) {
	// check exist firstly
	exist, err := db.Query(`SELECT id FROM coin WHERE account_id = ? AND block_id = ?AND transaction_id = ? AND output_index = ?`, accountID, blockHash, txID, index)
	if err != nil {
		return -1, err
	}
	defer exist.Close()
	if exist.Next() {
		var id int64
		err := exist.Scan(&id)
		if err != nil {
			return -1, err
		}
		return id, err
	}
	stmt, err := db.Prepare(`INSERT INTO coin (account_id,transaction_version,transaction_id,output_index ,coin_value,block_id ,block_height ,is_coinbase,status,data) 
									VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return -1, err
	}
	result, err := stmt.Exec(
		accountID,
		txVersion,
		txID,
		index,
		value,
		blockHash,
		blockHeight,
		isCoinbase,
		0,
		data)
	if err != nil {
		return -1, err
	}
	return result.LastInsertId()
}
func LoadImmatureCoinbaseCoins(height int64) ([]*Coin, error) {
	rows, err := db.Query(`
SELECT ID,account_id,transaction_id,output_index,coin_value,block_id ,block_height 
FROM coin  
WHERE is_coinbase=True AND status = 0 AND block_height < ?`, height)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coins := make([]*Coin, 0)
	for rows.Next() {
		var id int64
		var accountID int64
		var txVersion uint32
		var txID string
		var outputIndex uint8
		var value int64
		var blockID string
		var blockHeight int64

		err = rows.Scan(&id,
			&id,
			&accountID,
			&txID,
			&outputIndex,
			&value,
			&blockID,
			&blockHeight)
		if err != nil {
			return nil, err
		}
		coins = append(coins, &Coin{
			ID:        id,
			AccountID: accountID,
			Coin: abelian.NewCoin(txVersion, txID, outputIndex,
				blockID, blockHeight, value, "", nil),
		})
	}
	return coins, err
}

func LoadImmatureCoins(height int64) ([]*Coin, error) {
	rows, err := db.Query(`
SELECT ID,account_id,transaction_id,output_index,coin_value,block_id ,block_height 
FROM coin  
WHERE status = 0 AND block_height <= ?`, height)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coins := make([]*Coin, 0)
	for rows.Next() {
		var id int64
		var accountID int64
		var txVersion uint32
		var txID string
		var outputIndex uint8
		var value int64
		var blockID string
		var blockHeight int64

		err = rows.Scan(
			&id,
			&accountID,
			&txID,
			&outputIndex,
			&value,
			&blockID,
			&blockHeight)
		if err != nil {
			return nil, err
		}
		coins = append(coins, &Coin{
			ID:        id,
			AccountID: accountID,
			Coin: abelian.NewCoin(txVersion, txID, outputIndex,
				blockID, blockHeight, value, "", nil),
		})
	}
	return coins, nil
}
func LoadCoinByAccountID(id int64) ([]*Coin, error) {
	rows, err := db.Query(`SELECT ID,account_id,transaction_version,transaction_id,output_index,coin_value,block_id ,block_height, data
								 FROM coin  
								WHERE account_id = ? AND status = 1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coins := make([]*Coin, 0)
	for rows.Next() {
		var ID int64
		var accountID int64
		var txVersion uint32
		var txID string
		var outputIndex uint8
		var value int64
		var blockID string
		var blockHeight int64
		var data []byte

		err = rows.Scan(
			&ID,
			&accountID,
			&txVersion,
			&txID,
			&outputIndex,
			&value,
			&blockID,
			&blockHeight,
			&data)
		if err != nil {
			return nil, err
		}
		coins = append(coins, &Coin{
			ID:        id,
			AccountID: accountID,
			Coin: abelian.NewCoin(txVersion, txID, outputIndex,
				blockID, blockHeight, value, "", data),
		})
	}
	return coins, err
}
func LoadCoinBySerialNumber(serialNumber string) ([]*Coin, error) {
	rows, err := db.Query(`
SELECT ID,account_id,transaction_id,output_index,coin_value,block_id ,block_height 
FROM coin  
WHERE serial_number = ?`, serialNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coins := make([]*Coin, 0)
	for rows.Next() {
		var id int64
		var accountID int64
		var txVersion uint32
		var txID string
		var outputIndex uint8
		var value int64
		var blockID string
		var blockHeight int64

		err = rows.Scan(
			&id,
			&accountID,
			&txID,
			&outputIndex,
			&value,
			&blockID,
			&blockHeight)
		if err != nil {
			return nil, err
		}
		coins = append(coins, &Coin{
			ID:        id,
			AccountID: accountID,
			Coin: abelian.NewCoin(txVersion, txID, outputIndex,
				blockID, blockHeight, value, "", nil),
		})
	}
	return coins, err
}

func UpdateSerialNumber(id int64, serialNumber string) error {
	stmt, err := db.Prepare("UPDATE coin SET serial_number = ? WHERE id = ?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(
		serialNumber,
		id)
	return err
}

func updateCoinStatus(id int64, status int) error {
	stmt, err := db.Prepare("UPDATE coin SET status = ? WHERE id = ?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(
		status,
		id)
	return err
}

func MaturesCoin(id int64) error {
	return updateCoinStatus(id, 1)
}
func SpendCoin(id int64) error {
	return updateCoinStatus(id, 2)
}

func ConfirmSpentCoin(id int64) error {
	return updateCoinStatus(id, 3)
}
