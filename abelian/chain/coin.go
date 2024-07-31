package chain

import (
	"fmt"
)

type CoinID struct {
	TxID  string
	Index uint8
}

// Define methods for CoinID.
func NewCoinID(txHash string, index uint8) *CoinID {
	return &CoinID{
		TxID:  txHash,
		Index: index,
	}
}

func (id CoinID) String() string {
	return fmt.Sprintf("%s:%d", id.TxID, id.Index)
}

type Coin struct {
	ID           CoinID
	OwnerAddress *AbelAddress
	Value        int64
	SerialNumber []byte
	TxVoutData   []byte
	BlockHash    []byte
	BlockHeight  int64
}
