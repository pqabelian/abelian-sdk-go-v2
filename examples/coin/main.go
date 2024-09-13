package main

import (
	"bytes"
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
						coin.AccountID, coin.BlockHash, coin.BlockHeight, tx.TxHash, index, abelian.NeutrinoToAbel(coin.Value))

					break
				}
			}

		}
	}
	return nil
}
func BuildCoinRingsWithBlockHeight(height int64) (map[abelian.CoinID]*abelian.CoinRing, error) {
	serializedBlockGroups, err := abelian.GetRingBlockGroupByHeight(client, height)
	if err != nil {
		return nil, fmt.Errorf("fail to get block group with height %d: %v", height, err)
	}

	coinIDRings, err := abelian.BuildCoinRings(serializedBlockGroups)
	if err != nil {
		return nil, fmt.Errorf("fail to build txo rings with height %d: %v", height, err)
	}
	// build mapping: coin id -> ring
	coinID2RingDetail := map[abelian.CoinID]*abelian.CoinRing{}
	for i := 0; i < len(coinIDRings); i++ {
		ring := coinIDRings[i]
		for j := 0; j < len(ring.CoinIDRing.CoinIDs); j++ {
			coinID2RingDetail[*ring.CoinIDRing.CoinIDs[j]] = ring
		}
	}
	return coinID2RingDetail, nil
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
		err = database.MatureCoin(coin.ID)
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

	// NOTE: the height that meets the above conditions should mature some coins, here show for specified height
	// how to handle maturity with ring
	coinID2CoinRing, err := BuildCoinRingsWithBlockHeight(height)
	if err != nil {
		return err
	}

	// in this point, some coins would be mature
	immatureCoins, err := database.LoadImmatureCoins(height)
	if err != nil {
		panic(fmt.Errorf("fail to load immature coins from database"))
	}
	for i := 0; i < len(immatureCoins); i++ {
		coin := immatureCoins[i]
		coinID := coin.Coin.ID()
		// Before mark coin mature, build rings from blocks ith height
		coinRing, ok := coinID2CoinRing[*coinID]
		if !ok {
			return fmt.Errorf("can not found ring detail for coin (%s)", coinID.String())
		}

		ringIndex := 0
		for ; ringIndex < len(coinRing.CoinIDRing.CoinIDs); ringIndex++ {
			if coinRing.CoinIDRing.CoinIDs[ringIndex].TxID == coinID.TxID &&
				coinRing.CoinIDRing.CoinIDs[ringIndex].Index == coinID.Index {
				break
			}
		}

		ringId, err := coinRing.RingId()
		if err != nil {
			panic(err)
		}
		coin.Coin.SetRingInfo(ringId, uint8(ringIndex))

		// generate serial number
		w := bytes.Buffer{}
		err = coinRing.Serialize(&w)
		if err != nil {
			panic(err)
		}
		serializedRing := w.Bytes()
		viewAccount, _ := database.LoadViewAccount(coin.AccountID)
		serialNumber, err := viewAccount.GenerateSerialNumberWithRing(
			&abelian.CoinID{
				TxID:  coin.TxID,
				Index: coin.Index,
			},
			serializedRing)
		if err != nil {
			panic(err)
		}

		// update the coin info
		err = database.UpdateCoinInfo(coin.ID, ringId, uint8(ringIndex), hex.EncodeToString(serialNumber))
		if err != nil {
			panic(fmt.Errorf("fail to update coins"))
		}

		// insert relevant coins which would be used when generating transaction to consume matured coins
		for j := 0; j < len(coinRing.CoinIDRing.CoinIDs); j++ {
			relevantCoinID := coinRing.CoinIDRing.CoinIDs[j]
			relevantSerializedTXO := coinRing.SerializedTxOuts[j]
			_, err = database.InsertRelevantCoin(0, coin.TxVersion, relevantCoinID.TxID, relevantCoinID.Index, "", 0, -1, coinRing.IsCoinbase, relevantSerializedTXO, ringId, uint8(j))
			if err != nil {
				panic(err)
			}
		}

		// insert ring which would be used when generating transaction to consume matured coins
		_, err = database.InsertRing(
			ringId,
			coinRing.Version,
			int64(coinRing.RingBlockHeight),
			coinRing.CoinIDRing.BlockIDs,
			int8(len(coinRing.CoinIDRing.CoinIDs)),
			coinRing.IsCoinbase)
		if err != nil {
			panic(err)
		}
		fmt.Printf("📢 Ring related for coin (%s,%d) %v ABELs is inserted\n", coin.TxID, coin.Index, abelian.NeutrinoToAbel(coin.Value))

		// lastly mark coin as mature
		err = database.MatureCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mature immature coins"))
		}
		fmt.Printf("🎉 transfer coin (%s,%d) %v ABELs mature with ring info (%s,%d) \n", coin.TxID, coin.Index, abelian.NeutrinoToAbel(coin.Value), ringId, ringIndex)
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
