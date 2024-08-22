package main

import (
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func main() {
	client := common.Client

	startHeight := common.GetTxTrackStartHeight()
	endHeight := common.GetTxTrackEndHeight()

	pendingTxs, err := database.LoadPendingTransactions()
	if err != nil {
		panic(err)
	}
	for currentHeight := startHeight; currentHeight < endHeight; currentHeight++ {
		block, err := client.GetBlockByHeight(currentHeight)
		if err != nil {
			panic(fmt.Errorf("fail to query block: %v", err))
		}

		fmt.Printf("track transactions in block with height %d\n", block.Height)
		for i := 0; i < len(block.TxHashes); i++ {
			for _, pendingTx := range pendingTxs {
				if block.TxHashes[i] == pendingTx.TxID {
					fmt.Printf("Transaction %s confirmed at height %d \n", pendingTx.TxID, block.Height)
					err = database.MarkTxConfirmed(pendingTx.TxID)
					if err != nil {
						panic(err)
					}
					// Also mark coins as confirmed which is done in examples/coin/main.go
				}
			}
		}
	}
	return
}
