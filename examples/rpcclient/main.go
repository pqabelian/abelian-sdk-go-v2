package main

import (
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
)

func main() {
	client := common.Client

	info, err := client.GetChainInfo()
	if err != nil {
		panic(fmt.Errorf("fail to get block hash: %v", err))
	}
	fmt.Printf("chain info: %#+v\n", info)

	height := int32(0)
	blockID, err := client.GetBlockHash(height)
	if err != nil {
		panic(fmt.Errorf("fail to get block id: %v", err))
	}
	fmt.Printf("block hash is %s at height %d\n", blockID, height)

	block, err := client.GetBlock(blockID)
	if err != nil {
		panic(fmt.Errorf("fail to get block id: %v", err))
	}
	fmt.Printf("block with id %s: %#+v\n", blockID, block)

	tx, err := client.GetRawTx(block.TxHashes[0])
	if err != nil {
		panic(fmt.Errorf("fail to get transaction: %v", err))
	}
	fmt.Printf("tx with id %s: %#+v\n", block.TxHashes[0], tx)

	unconfirmedTxs, err := client.GetRawMempool()
	if err != nil {
		panic(fmt.Errorf("fail to get mempool: %v", err))
	}
	if len(unconfirmedTxs) == 0 {
		fmt.Printf("mempool has no unconfirmed transactions.\n")
	} else {
		fmt.Printf("mempool has %d unconfirmed transactions:\n", len(unconfirmedTxs))
		for i := 0; i < len(unconfirmedTxs); i++ {
			fmt.Printf("%s\n", unconfirmedTxs[i])
		}
	}
}
