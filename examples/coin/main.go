package main

import (
	"encoding/hex"
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func ScanCoins(viewAccounts []*database.ViewAccount, tx *abelian.Tx, isCoinbaseTx bool, blockID string, blockHeight int64) error {
	for index := 0; index < len(tx.Vout); index++ {
		txOutData, err := hex.DecodeString(tx.Vout[index].Script)
		if err != nil {
			return fmt.Errorf("fail to decode output of transaction: %v", err)
		}
		for _, viewAccount := range viewAccounts {
			success, value, err := viewAccount.ReceiveCoin(uint32(tx.Version), txOutData)
			if err != nil {
				return fmt.Errorf("fail to decode output of transaction: %v", err)
			}
			if !success {
				continue
			}

			fmt.Printf("💰 Find coin of account with account id %d: block id %s, block height %d, transacion id %s, index %d, value %v ABELs\n",
				viewAccount.ID, blockID, blockHeight, tx.TxHash, index, abelian.NeutrinoToAbel(int64(value)))

			_, err = database.InsertCoin(viewAccount.ID, tx.Version, tx.TxID, uint8(index), blockID, blockHeight, int64(value), isCoinbaseTx, txOutData)
			if err != nil {
				return fmt.Errorf("fail to store coin into database: %v", err)
			}
		}
	}
	return nil
}

func TrackCoins(tx *abelian.Tx) error {
	for index := 0; index < len(tx.Vin); index++ {
		mayConsumedCoins, err := database.LoadCoinBySerialNumber(tx.Vin[index].SerialNumber)
		if err != nil {
			panic(fmt.Errorf("fail to load coin by serial number from database: %v", err))
		}

		for _, coin := range mayConsumedCoins {
			coinID := coin.Coin.ID()
			for ringIndex := 0; ringIndex < len(tx.Vin[index].TXORing.OutPoints); ringIndex++ {
				if coinID.TxID == tx.Vin[index].TXORing.OutPoints[ringIndex].TxHash &&
					coinID.Index == tx.Vin[index].TXORing.OutPoints[ringIndex].Index {
					err = database.ConfirmSpentCoin(coin.ID)
					if err != nil {
						panic(fmt.Errorf("fail to consume coin: %v", err))
					}

					fmt.Printf("💸 Coin of account with account id %d is consumed: block id %s, block height %d, transacion id %s, index %d, value %v ABELs\n",
						coin.ID, coin.BlockHash, coin.BlockHeight, tx.TxHash, index, abelian.NeutrinoToAbel(coin.Value))

					break
				}
			}

		}
	}
	return nil
}

func HandleCoinMaturity(height int64) error {
	// handle coinbase coin maturity
	fmt.Printf("handle coinbase maturity in block with height %d \n", height)
	immatureCoinbaseCoins, err := database.LoadImmatureCoinbaseCoins(height - abelian.GetCoinbaseMaturity())
	if err != nil {
		panic(fmt.Errorf("fail to load immature coinbase coins from database"))
	}
	for _, coin := range immatureCoinbaseCoins {
		fmt.Printf("🎉 coinbase coin (%s,%d) %v ABELs is mature\n", coin.TxID, coin.Index, abelian.NeutrinoToAbel(coin.Value))
		err = database.MaturesCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mature immature coinbase coins"))
		}
	}

	// handle transfer coins maturity
	blockNum := int64(abelian.GetBlockNumPerRingGroupByBlockHeight(height))
	if height%blockNum != blockNum-1 {
		return nil
	}
	fmt.Printf("handle transfer maturity in block with height %d \n", height)
	// in this point, some coin would be mature
	immatureCoins, err := database.LoadImmatureCoins(height)
	if err != nil {
		panic(fmt.Errorf("fail to load immature coins from database"))
	}
	for i := 0; i < len(immatureCoins); i++ {
		coin := immatureCoins[i]
		viewAccount, _ := database.LoadViewAccount(coin.AccountID)
		serializedBlockGroups, err := abelian.GetRingBlockGroupByHeight(client, coin.BlockHeight)
		if err != nil {
			panic(err)
		}
		serialNumber, err := viewAccount.GenerateSerialNumberWithBlocks(
			&abelian.CoinID{
				TxID:  coin.TxID,
				Index: coin.Index,
			},
			serializedBlockGroups)
		if err != nil {
			panic(err)
		}

		// update serial number of coin
		err = database.UpdateSerialNumber(coin.ID, hex.EncodeToString(serialNumber))
		if err != nil {
			panic(err)
		}

		// meanwhile, mark the coin mature
		err = database.MaturesCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mature immature coinbase coins"))
		}
		fmt.Printf("🎉 transfer coin (%s,%d) %v ABELs is mature\n", coin.TxID, coin.Index, abelian.NeutrinoToAbel(coin.Value))
	}

	return nil
}

var client *abelian.Client

func init() {
	client = common.Client
}

func main() {
	// Load view accounts from database
	viewAccounts, err := database.LoadViewAccounts()
	if err != nil {
		panic(err)
	}

	if len(viewAccounts) == 0 {
		panic(fmt.Errorf("no view account found in database"))
	}

	// Scan and store coins for accounts with specified height ranges
	startScanHeight := common.GetCoinScanStartHeight()
	endScanHeight := common.GetCoinScanEndHeight()

	// scan blocks with specified height scope
	for currentHeight := startScanHeight; currentHeight < endScanHeight; currentHeight++ {
		block, err := client.GetBlockByHeight(currentHeight)
		if err != nil {
			panic(fmt.Errorf("fail to query block: %v", err))
		}

		fmt.Printf("Scan and track coins in block with height %d\n", block.Height)
		for i := 0; i < len(block.TxHashes); i++ {
			tx, err := client.GetRawTx(block.TxHashes[i])
			if err != nil {
				panic(fmt.Errorf("fail to query transaction: %v", err))
			}

			// scan coin in transaction
			err = ScanCoins(viewAccounts, tx, i == 0, block.BlockHash, block.Height)
			if err != nil {
				panic(fmt.Errorf("fail to scan tranaction:%v", err))
			}

			// track coin status
			// for coinbase transaction, there no valid consumption
			if i == 0 {
				continue
			}
			// but for transfer transaction, There must be valid consumption
			err = TrackCoins(tx)
			if err != nil {
				panic(fmt.Errorf("fail to track coin status :%v", err))
			}
		}

		err = HandleCoinMaturity(currentHeight)
		if err != nil {
			panic(fmt.Errorf("fail to hanle coin maturity :%v", err))
		}
	}
}
