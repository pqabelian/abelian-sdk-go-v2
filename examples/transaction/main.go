package main

import (
	"encoding/hex"
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
	"sort"
)

func SignRawTransaction(unsignedRawTx *abelian.UnsignedRawTx, senderAccountIDs []int64) (*abelian.SignedRawTx, error) {
	// Load account (NOT view account!!!)
	senderAccounts := make([]abelian.Account, 0, len(senderAccountIDs))
	for _, accountID := range senderAccountIDs {
		account, err := database.LoadAccountByID(accountID)
		if err != nil {
			panic(fmt.Errorf("fail to load account: %v", err))
		}
		senderAccounts = append(senderAccounts, account.Account)
	}

	// Sign the unsigned transaction
	return abelian.GenerateSignedRawTx(unsignedRawTx, senderAccounts)
}

var client *abelian.Client

func init() {
	client = common.Client
}

func main() {
	// Specify the transfer addresses and amounts, and specify change address
	// Here we fetch built-in account with id 3 & 4.
	accountForFullPrivacy, err := database.LoadAccountByID(3)
	if err != nil {
		panic("fail to load account with id 3")
	}
	accountForPseudonymous, err := database.LoadAccountByID(4)
	if err != nil {
		panic("fail to load account with id 4")
	}
	// And then generated address
	changeAddress, err := accountForPseudonymous.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account with id 3")
	}

	receiverAddresses := make([]string, 2)
	tmpAbelAddress, err := accountForFullPrivacy.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated address for account with id 3")
	}
	receiverAddresses[0] = hex.EncodeToString(tmpAbelAddress)

	tmpAbelAddress, err = accountForPseudonymous.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated address for account with id 4")
	}
	receiverAddresses[1] = hex.EncodeToString(tmpAbelAddress)

	receiverInfos := []struct {
		AbelAddress string
		CoinValue   int64
	}{
		{
			AbelAddress: receiverAddresses[1],
			CoinValue:   20_0000000,
		},
		{
			AbelAddress: receiverAddresses[0],
			CoinValue:   30_0000000,
		},
	}
	targetValue := int64(0)
	txOutDescs := make([]*abelian.TxOutDesc, len(receiverInfos))
	for i, info := range receiverInfos {
		abelAddress, err := hex.DecodeString(info.AbelAddress)
		if err != nil {
			panic("invalid abel address")
		}
		address, err := abelian.NewAbelAddress(abelAddress)
		if err != nil {
			panic("invalid abel address")
		}
		if address.GetNetID() != common.GetNetworkID() {
			panic("abel address with unmatched network id")
		}

		txOutDescs[i] = &abelian.TxOutDesc{
			AbelAddress: address,
			CoinValue:   info.CoinValue,
		}
		targetValue += info.CoinValue
	}

	// Load coins of specified account
	selectAccountIDs := []int64{3, 4}
	availableCoins := []*database.Coin{}
	for _, accountID := range selectAccountIDs {
		coins, err := database.LoadCoinByAccountID(accountID)
		if err != nil {
			return
		}
		availableCoins = append(availableCoins, coins...)
	}
	if len(availableCoins) == 0 {
		panic(fmt.Errorf("find no coin to spend"))
	}

	// Customized filters to select coins
	sort.SliceStable(availableCoins, func(i, j int) bool {
		return availableCoins[i].Value < availableCoins[j].Value
	})

	selectedCoins := []*database.Coin{}
	selectValue := int64(0)
	for i := 0; i < len(availableCoins); i++ {
		if selectValue >= targetValue {
			break
		}
		selectedCoins = append(selectedCoins, availableCoins[i])
		selectValue += availableCoins[i].Value
	}

	// Build TxInDesc
	txInDescs := []*abelian.TxInDesc{}
	blockGroups := map[int64]*abelian.TxBlockDesc{}
	coin2AccountID := map[string]int64{}
	for _, coin := range selectedCoins {
		txInDescs = append(txInDescs, &abelian.TxInDesc{
			BlockHeight: coin.BlockHeight,
			BlockID:     coin.BlockHash,
			TxVersion:   coin.TxVersion,
			TxID:        coin.TxID,
			TxOutIndex:  coin.Index,
			TxOutData:   coin.TxVoutData,
			CoinValue:   coin.Value,
		})
		blockHeights := abelian.GetRingBlockHeights(coin.BlockHeight)
		for _, height := range blockHeights {
			if _, ok := blockGroups[height]; !ok {
				blockBytes, err := client.GetBlockBytesByHeight(height)
				if err != nil {
					panic(fmt.Errorf("fail to get block group: %v", err))
				}
				blockGroups[height] = abelian.NewTxBlockDesc(blockBytes, height)
			}
		}
		coin2AccountID[coin.Coin.ID().String()] = coin.AccountID
	}

	// [IMPORTANT] Sort TxInDesc
	err = abelian.SortTxInDescs(txInDescs)
	if err != nil {
		panic(err)
	}

	// set sender account ID for signing
	senderAccountIDs := make([]int64, 0, len(txInDescs))
	for _, desc := range txInDescs {
		senderAccountIDs = append(senderAccountIDs, coin2AccountID[abelian.NewCoinID(desc.TxID, desc.TxOutIndex).String()])
	}

	// Estimated fee
	estimatedTxFee := abelian.EstimateTxFee(txInDescs, txOutDescs)

	// change if needed
	if selectValue-targetValue-estimatedTxFee > 0 {
		changeAbelAddress, err := abelian.NewAbelAddress(changeAddress)
		if err != nil {
			panic("invalid abel address")
		}
		if changeAbelAddress.GetNetID() != common.GetNetworkID() {
			panic("change address with unmatched network id")
		}

		txOutDescs = append(txOutDescs, &abelian.TxOutDesc{
			AbelAddress: changeAbelAddress,
			CoinValue:   selectValue - targetValue - estimatedTxFee,
		})
	}

	// [IMPORTANT] sort txOutDescs
	err = abelian.SortTxOutDesc(txOutDescs)
	if err != nil {
		panic(err)
	}

	//  Make an unsigned transaction
	txDesc := abelian.NewTxDesc(txInDescs, txOutDescs, estimatedTxFee, blockGroups)
	unsignedRawTx, err := abelian.GenerateUnsignedRawTx(txDesc)
	if err != nil {
		panic(fmt.Errorf("fail to generate unsigned raw tx: %v", err))
	}
	fmt.Println(unsignedRawTx)

	// Sign transaction
	signedRawTx, err := SignRawTransaction(unsignedRawTx, senderAccountIDs)
	if err != nil {
		panic(err)
	}
	// Broadcast signed transaction
	returnedTxHash, err := client.SendRawTx(hex.EncodeToString(signedRawTx.Data))
	if err != nil {
		panic(fmt.Errorf("fail to send raw tx: %v", err))
	}

	// assert equal
	if returnedTxHash != signedRawTx.TxID {
		panic(fmt.Errorf("unmatched tx id"))
	}
	fmt.Println("Submit transaction: ", returnedTxHash)

	_, err = database.InsertTx(returnedTxHash, 0, hex.EncodeToString(unsignedRawTx.Data), senderAccountIDs, hex.EncodeToString(signedRawTx.Data))
	if err != nil {
		panic(err)
	}
	// mark coin spent
	for _, coin := range selectedCoins {
		err = database.SpendCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mark coin spent: %v", err))
		}
	}
}
